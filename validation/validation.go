package validation

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
	"sandbox-api-go/errors"
)

// ValidationRule represents a validation rule
type ValidationRule func(value interface{}) *errors.ValidationError

// Validator handles input validation
type Validator struct {
	errors []errors.ValidationError
}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{
		errors: make([]errors.ValidationError, 0),
	}
}

// HasErrors returns true if there are validation errors
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// GetErrors returns all validation errors
func (v *Validator) GetErrors() []errors.ValidationError {
	return v.errors
}

// GetError returns a formatted AppError if there are validation errors
func (v *Validator) GetError() *errors.AppError {
	if !v.HasErrors() {
		return nil
	}
	return errors.NewValidationError(v.errors)
}

// addError adds a validation error
func (v *Validator) addError(field, message string, value interface{}) {
	v.errors = append(v.errors, errors.ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// ValidateField validates a field with multiple rules
func (v *Validator) ValidateField(field string, value interface{}, rules ...ValidationRule) *Validator {
	for _, rule := range rules {
		if err := rule(value); err != nil {
			err.Field = field
			if err.Value == nil {
				err.Value = value
			}
			v.errors = append(v.errors, *err)
		}
	}
	return v
}

// Common validation rules

// Required validates that a value is not empty
func Required() ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		if value == nil {
			return &errors.ValidationError{
				Message: "This field is required",
			}
		}
		
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) == "" {
				return &errors.ValidationError{
					Message: "This field is required",
				}
			}
		case *string:
			if v == nil || strings.TrimSpace(*v) == "" {
				return &errors.ValidationError{
					Message: "This field is required",
				}
			}
		}
		
		return nil
	}
}

// MinLength validates minimum string length
func MinLength(min int) ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		str, ok := value.(string)
		if !ok {
			return &errors.ValidationError{
				Message: "Value must be a string",
			}
		}
		
		if len(strings.TrimSpace(str)) < min {
			return &errors.ValidationError{
				Message: fmt.Sprintf("Must be at least %d characters long", min),
			}
		}
		
		return nil
	}
}

// MaxLength validates maximum string length
func MaxLength(max int) ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		str, ok := value.(string)
		if !ok {
			return &errors.ValidationError{
				Message: "Value must be a string",
			}
		}
		
		if len(str) > max {
			return &errors.ValidationError{
				Message: fmt.Sprintf("Must be no more than %d characters long", max),
			}
		}
		
		return nil
	}
}

// Email validates email format
func Email() ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		str, ok := value.(string)
		if !ok {
			return &errors.ValidationError{
				Message: "Value must be a string",
			}
		}
		
		str = strings.TrimSpace(str)
		if str == "" {
			return nil // Let Required() handle empty values
		}
		
		_, err := mail.ParseAddress(str)
		if err != nil {
			return &errors.ValidationError{
				Message: "Must be a valid email address",
			}
		}
		
		return nil
	}
}

// Username validates username format
func Username() ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		str, ok := value.(string)
		if !ok {
			return &errors.ValidationError{
				Message: "Value must be a string",
			}
		}
		
		str = strings.TrimSpace(str)
		if str == "" {
			return nil // Let Required() handle empty values
		}
		
		// Username rules: 3-30 chars, alphanumeric and underscore, must start with letter
		if len(str) < 3 || len(str) > 30 {
			return &errors.ValidationError{
				Message: "Username must be between 3 and 30 characters",
			}
		}
		
		// Must start with a letter
		if !unicode.IsLetter(rune(str[0])) {
			return &errors.ValidationError{
				Message: "Username must start with a letter",
			}
		}
		
		// Only letters, numbers, and underscores
		validUsername := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
		if !validUsername.MatchString(str) {
			return &errors.ValidationError{
				Message: "Username can only contain letters, numbers, and underscores",
			}
		}
		
		return nil
	}
}

// Password validates password strength
func Password() ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		str, ok := value.(string)
		if !ok {
			return &errors.ValidationError{
				Message: "Value must be a string",
			}
		}
		
		if str == "" {
			return nil // Let Required() handle empty values
		}
		
		// Password strength rules
		if len(str) < 8 {
			return &errors.ValidationError{
				Message: "Password must be at least 8 characters long",
			}
		}
		
		if len(str) > 128 {
			return &errors.ValidationError{
				Message: "Password must be no more than 128 characters long",
			}
		}
		
		var hasUpper, hasLower, hasNumber bool
		for _, char := range str {
			switch {
			case unicode.IsUpper(char):
				hasUpper = true
			case unicode.IsLower(char):
				hasLower = true
			case unicode.IsNumber(char):
				hasNumber = true
			}
		}
		
		if !hasUpper {
			return &errors.ValidationError{
				Message: "Password must contain at least one uppercase letter",
			}
		}
		
		if !hasLower {
			return &errors.ValidationError{
				Message: "Password must contain at least one lowercase letter",
			}
		}
		
		if !hasNumber {
			return &errors.ValidationError{
				Message: "Password must contain at least one number",
			}
		}
		
		return nil
	}
}

// NotEmpty validates that a string is not empty (different from Required - this trims whitespace)
func NotEmpty() ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		str, ok := value.(string)
		if !ok {
			return &errors.ValidationError{
				Message: "Value must be a string",
			}
		}
		
		if strings.TrimSpace(str) == "" {
			return &errors.ValidationError{
				Message: "This field cannot be empty",
			}
		}
		
		return nil
	}
}

// Range validates that a number is within a range
func Range(min, max int) ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		var num int
		switch v := value.(type) {
		case int:
			num = v
		case *int:
			if v == nil {
				return nil
			}
			num = *v
		default:
			return &errors.ValidationError{
				Message: "Value must be a number",
			}
		}
		
		if num < min || num > max {
			return &errors.ValidationError{
				Message: fmt.Sprintf("Value must be between %d and %d", min, max),
			}
		}
		
		return nil
	}
}

// OneOf validates that a value is one of the allowed values
func OneOf(allowed ...interface{}) ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		for _, allowedValue := range allowed {
			if value == allowedValue {
				return nil
			}
		}
		
		return &errors.ValidationError{
			Message: fmt.Sprintf("Value must be one of: %v", allowed),
		}
	}
}

// Custom validation functions for models

// ValidateRegisterRequest validates user registration input
func ValidateRegisterRequest(username, email, password string) *errors.AppError {
	validator := NewValidator()
	
	validator.ValidateField("username", username, Required(), Username())
	validator.ValidateField("email", email, Required(), Email())
	validator.ValidateField("password", password, Required(), Password())
	
	return validator.GetError()
}

// ValidateLoginRequest validates user login input
func ValidateLoginRequest(email, password string) *errors.AppError {
	validator := NewValidator()
	
	validator.ValidateField("email", email, Required(), Email())
	validator.ValidateField("password", password, Required())
	
	return validator.GetError()
}

// ValidateTaskInput validates task creation/update input
func ValidateTaskInput(title, description string) *errors.AppError {
	validator := NewValidator()
	
	validator.ValidateField("title", title, Required(), NotEmpty(), MaxLength(200))
	validator.ValidateField("description", description, MaxLength(1000))
	
	return validator.GetError()
}