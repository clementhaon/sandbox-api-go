package validation

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"

	"github.com/clementhaon/sandbox-api-go/pkg/errors"
)

type ValidationRule func(value interface{}) *errors.ValidationError

type Validator struct {
	errors []errors.ValidationError
}

func NewValidator() *Validator {
	return &Validator{errors: make([]errors.ValidationError, 0)}
}

func (v *Validator) HasErrors() bool                    { return len(v.errors) > 0 }
func (v *Validator) GetErrors() []errors.ValidationError { return v.errors }

func (v *Validator) GetError() *errors.AppError {
	if !v.HasErrors() {
		return nil
	}
	return errors.NewValidationError(v.errors)
}

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

func Required() ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		if value == nil {
			return &errors.ValidationError{Message: "This field is required"}
		}
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) == "" {
				return &errors.ValidationError{Message: "This field is required"}
			}
		case *string:
			if v == nil || strings.TrimSpace(*v) == "" {
				return &errors.ValidationError{Message: "This field is required"}
			}
		}
		return nil
	}
}

func MinLength(min int) ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		str, ok := value.(string)
		if !ok {
			return &errors.ValidationError{Message: "Value must be a string"}
		}
		if len(strings.TrimSpace(str)) < min {
			return &errors.ValidationError{Message: fmt.Sprintf("Must be at least %d characters long", min)}
		}
		return nil
	}
}

func MaxLength(max int) ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		str, ok := value.(string)
		if !ok {
			return &errors.ValidationError{Message: "Value must be a string"}
		}
		if len(str) > max {
			return &errors.ValidationError{Message: fmt.Sprintf("Must be no more than %d characters long", max)}
		}
		return nil
	}
}

func Email() ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		str, ok := value.(string)
		if !ok {
			return &errors.ValidationError{Message: "Value must be a string"}
		}
		str = strings.TrimSpace(str)
		if str == "" {
			return nil
		}
		if _, err := mail.ParseAddress(str); err != nil {
			return &errors.ValidationError{Message: "Must be a valid email address"}
		}
		return nil
	}
}

func Username() ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		str, ok := value.(string)
		if !ok {
			return &errors.ValidationError{Message: "Value must be a string"}
		}
		str = strings.TrimSpace(str)
		if str == "" {
			return nil
		}
		if len(str) < 3 || len(str) > 30 {
			return &errors.ValidationError{Message: "Username must be between 3 and 30 characters"}
		}
		if !unicode.IsLetter(rune(str[0])) {
			return &errors.ValidationError{Message: "Username must start with a letter"}
		}
		validUsername := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
		if !validUsername.MatchString(str) {
			return &errors.ValidationError{Message: "Username can only contain letters, numbers, and underscores"}
		}
		return nil
	}
}

func Password() ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		str, ok := value.(string)
		if !ok {
			return &errors.ValidationError{Message: "Value must be a string"}
		}
		if str == "" {
			return nil
		}
		if len(str) < 8 {
			return &errors.ValidationError{Message: "Password must be at least 8 characters long"}
		}
		if len(str) > 128 {
			return &errors.ValidationError{Message: "Password must be no more than 128 characters long"}
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
			return &errors.ValidationError{Message: "Password must contain at least one uppercase letter"}
		}
		if !hasLower {
			return &errors.ValidationError{Message: "Password must contain at least one lowercase letter"}
		}
		if !hasNumber {
			return &errors.ValidationError{Message: "Password must contain at least one number"}
		}
		return nil
	}
}

func NotEmpty() ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		str, ok := value.(string)
		if !ok {
			return &errors.ValidationError{Message: "Value must be a string"}
		}
		if strings.TrimSpace(str) == "" {
			return &errors.ValidationError{Message: "This field cannot be empty"}
		}
		return nil
	}
}

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
			return &errors.ValidationError{Message: "Value must be a number"}
		}
		if num < min || num > max {
			return &errors.ValidationError{Message: fmt.Sprintf("Value must be between %d and %d", min, max)}
		}
		return nil
	}
}

func OneOf(allowed ...interface{}) ValidationRule {
	return func(value interface{}) *errors.ValidationError {
		for _, allowedValue := range allowed {
			if value == allowedValue {
				return nil
			}
		}
		return &errors.ValidationError{Message: fmt.Sprintf("Value must be one of: %v", allowed)}
	}
}

func ValidateRegisterRequest(username, email, password string) *errors.AppError {
	validator := NewValidator()
	validator.ValidateField("username", username, Required(), Username())
	validator.ValidateField("email", email, Required(), Email())
	validator.ValidateField("password", password, Required(), Password())
	return validator.GetError()
}

func ValidateLoginRequest(email, password string) *errors.AppError {
	validator := NewValidator()
	validator.ValidateField("email", email, Required(), Email())
	validator.ValidateField("password", password, Required())
	return validator.GetError()
}

func ValidateTaskInput(title, description string) *errors.AppError {
	validator := NewValidator()
	validator.ValidateField("title", title, Required(), NotEmpty(), MaxLength(200))
	validator.ValidateField("description", description, MaxLength(1000))
	return validator.GetError()
}
