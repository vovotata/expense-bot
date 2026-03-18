package handlers

import "testing"

func TestValidateAmount(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"12.452", true},
		{"100", true},
		{"0.000001", true},
		{"999999999999.123456", true},
		{"0", false},
		{"-1", false},
		{"abc", false},
		{"12.1234567", false}, // 7 decimal places
		{"", false},
		{"12.", false},
		{".5", false},
		{"12,5", false},
		{"1 000", false},
	}

	for _, tt := range tests {
		_, err := ValidateAmount(tt.input)
		if tt.valid && err != nil {
			t.Errorf("ValidateAmount(%q) = error %v, want valid", tt.input, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidateAmount(%q) = valid, want error", tt.input)
		}
	}
}

func TestValidateComment(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"оплатить за 10 мин", true},
		{"a", true},
		{"", false},
		{string(make([]byte, 501)), false},
		{string(make([]byte, 500)), true},
	}

	for _, tt := range tests {
		_, err := ValidateComment(tt.input)
		if tt.valid && err != nil {
			t.Errorf("ValidateComment(len=%d) = error, want valid", len(tt.input))
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidateComment(len=%d) = valid, want error", len(tt.input))
		}
	}
}

func TestValidateAddress(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"TRC20abc123def", true},
		{"", false},
		{string(make([]byte, 257)), false},
	}

	for _, tt := range tests {
		_, err := ValidateAddress(tt.input)
		if tt.valid && err != nil {
			t.Errorf("ValidateAddress(%q) unexpected error: %v", tt.input, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidateAddress(%q) expected error", tt.input)
		}
	}
}

func TestValidateAccount(t *testing.T) {
	_, err := ValidateAccount("")
	if err == nil {
		t.Error("expected error for empty account")
	}
	_, err = ValidateAccount("my_account")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
