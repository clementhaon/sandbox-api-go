package validation

import (
	"testing"
)

func TestValidateRegisterRequest(t *testing.T) {
	tests := []struct {
		name     string
		username string
		email    string
		password string
		wantErr  bool
	}{
		{"valid", "johndoe", "john@example.com", "Password1", false},
		{"empty username", "", "john@example.com", "Password1", true},
		{"empty email", "johndoe", "", "Password1", true},
		{"empty password", "johndoe", "john@example.com", "", true},
		{"invalid email", "johndoe", "not-an-email", "Password1", true},
		{"short username", "ab", "john@example.com", "Password1", true},
		{"username starts with number", "1john", "john@example.com", "Password1", true},
		{"weak password no uppercase", "johndoe", "john@example.com", "password1", true},
		{"weak password no lowercase", "johndoe", "john@example.com", "PASSWORD1", true},
		{"weak password no number", "johndoe", "john@example.com", "Password", true},
		{"short password", "johndoe", "john@example.com", "Pass1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegisterRequest(tt.username, tt.email, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRegisterRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateLoginRequest(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		wantErr  bool
	}{
		{"valid", "john@example.com", "password123", false},
		{"empty email", "", "password123", true},
		{"empty password", "john@example.com", "", true},
		{"invalid email", "not-an-email", "password123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLoginRequest(tt.email, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLoginRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTaskInput(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		description string
		wantErr     bool
	}{
		{"valid", "My Task", "A description", false},
		{"valid empty description", "My Task", "", false},
		{"empty title", "", "desc", true},
		{"whitespace title", "   ", "desc", true},
		{"title too long", string(make([]byte, 201)), "", true},
		{"description too long", "Valid Title", string(make([]byte, 1001)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskInput(tt.title, tt.description)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTaskInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequired(t *testing.T) {
	rule := Required()

	if rule(nil) == nil {
		t.Error("Expected error for nil value")
	}
	if rule("") == nil {
		t.Error("Expected error for empty string")
	}
	if rule("  ") == nil {
		t.Error("Expected error for whitespace string")
	}
	if rule("hello") != nil {
		t.Error("Expected no error for non-empty string")
	}
}

func TestMinLength(t *testing.T) {
	rule := MinLength(3)

	if rule("ab") == nil {
		t.Error("Expected error for string shorter than min")
	}
	if rule("abc") != nil {
		t.Error("Expected no error for string at min length")
	}
	if rule("abcd") != nil {
		t.Error("Expected no error for string longer than min")
	}
}

func TestMaxLength(t *testing.T) {
	rule := MaxLength(5)

	if rule("abcdef") == nil {
		t.Error("Expected error for string longer than max")
	}
	if rule("abcde") != nil {
		t.Error("Expected no error for string at max length")
	}
	if rule("abc") != nil {
		t.Error("Expected no error for string shorter than max")
	}
}

func TestEmail(t *testing.T) {
	rule := Email()

	if rule("") != nil {
		t.Error("Expected no error for empty string (Required handles that)")
	}
	if rule("test@example.com") != nil {
		t.Error("Expected no error for valid email")
	}
	if rule("not-an-email") == nil {
		t.Error("Expected error for invalid email")
	}
}

func TestPassword(t *testing.T) {
	rule := Password()

	if rule("") != nil {
		t.Error("Expected no error for empty string (Required handles that)")
	}
	if rule("Password1") != nil {
		t.Error("Expected no error for valid password")
	}
	if rule("pass") == nil {
		t.Error("Expected error for short password")
	}
	if rule("password1") == nil {
		t.Error("Expected error for no uppercase")
	}
	if rule("PASSWORD1") == nil {
		t.Error("Expected error for no lowercase")
	}
	if rule("Password") == nil {
		t.Error("Expected error for no number")
	}
}

func TestUsername(t *testing.T) {
	rule := Username()

	if rule("") != nil {
		t.Error("Expected no error for empty string")
	}
	if rule("john_doe") != nil {
		t.Error("Expected no error for valid username")
	}
	if rule("ab") == nil {
		t.Error("Expected error for short username")
	}
	if rule("1john") == nil {
		t.Error("Expected error for username starting with number")
	}
	if rule("john doe") == nil {
		t.Error("Expected error for username with space")
	}
}

func TestOneOf(t *testing.T) {
	rule := OneOf("low", "medium", "high")

	if rule("medium") != nil {
		t.Error("Expected no error for valid value")
	}
	if rule("invalid") == nil {
		t.Error("Expected error for invalid value")
	}
}

func TestRange(t *testing.T) {
	rule := Range(1, 10)

	if rule(5) != nil {
		t.Error("Expected no error for value in range")
	}
	if rule(0) == nil {
		t.Error("Expected error for value below range")
	}
	if rule(11) == nil {
		t.Error("Expected error for value above range")
	}

	var nilPtr *int
	if rule(nilPtr) != nil {
		t.Error("Expected no error for nil pointer")
	}
}
