package emailwatch

import (
	"strings"
	"testing"
)

func TestParseEmail_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		wantCode string
		wantRule string
		wantNil  bool
	}{
		{
			name: "Google verification code",
			email: "From: noreply@google.com\r\n" +
				"Subject: Verification code\r\n" +
				"Content-Type: text/plain\r\n" +
				"\r\n" +
				"Your verification code is 847291\r\n",
			wantCode: "847291",
			wantRule: "Google",
		},
		{
			name: "Facebook code with FB- prefix",
			email: "From: security@facebookmail.com\r\n" +
				"Subject: Login code\r\n" +
				"Content-Type: text/plain\r\n" +
				"\r\n" +
				"Your login code is FB-12345678\r\n",
			wantCode: "12345678",
			wantRule: "Meta/Facebook",
		},
		{
			name: "Generic code in body",
			email: "From: noreply@service.com\r\n" +
				"Subject: Your code\r\n" +
				"Content-Type: text/plain\r\n" +
				"\r\n" +
				"Your verification code: 5432\r\n",
			wantCode: "5432",
			wantRule: "Generic numeric code",
		},
		{
			name: "Code with OTP label",
			email: "From: noreply@example.com\r\n" +
				"Subject: OTP\r\n" +
				"Content-Type: text/plain\r\n" +
				"\r\n" +
				"Your OTP: 987654\r\n",
			wantCode: "987654",
			wantRule: "Generic numeric code",
		},
		{
			name: "Russian language code",
			email: "From: noreply@bank.ru\r\n" +
				"Subject: Код подтверждения\r\n" +
				"Content-Type: text/plain\r\n" +
				"\r\n" +
				"Ваш код: 1234\r\n",
			wantCode: "1234",
			wantRule: "Generic numeric code",
		},
		{
			name: "PIN code",
			email: "From: noreply@app.com\r\n" +
				"Subject: Your PIN\r\n" +
				"Content-Type: text/plain\r\n" +
				"\r\n" +
				"Your PIN: 9876\r\n",
			wantCode: "9876",
			wantRule: "Generic numeric code",
		},
		{
			name: "No code found",
			email: "From: friend@example.com\r\n" +
				"Subject: Hello!\r\n" +
				"Content-Type: text/plain\r\n" +
				"\r\n" +
				"Hey, how are you doing?\r\n",
			wantNil: true,
		},
		{
			name: "Regular person email with code-like text ignored",
			email: "From: john@company.com\r\n" +
				"Subject: Meeting notes\r\n" +
				"Content-Type: text/plain\r\n" +
				"\r\n" +
				"Your code: 123456\r\n",
			wantNil: true,
		},
		{
			name: "Code in subject line",
			email: "From: noreply@example.com\r\n" +
				"Subject: Your verification code: 654321\r\n" +
				"Content-Type: text/plain\r\n" +
				"\r\n" +
				"Please use the code from the subject.\r\n",
			wantCode: "654321",
			wantRule: "Generic numeric code",
		},
		{
			name: "HTML email with code",
			email: "From: noreply@service.com\r\n" +
				"Subject: Verification\r\n" +
				"Content-Type: text/html\r\n" +
				"\r\n" +
				"<html><body><p>Your verification code: <b>112233</b></p></body></html>\r\n",
			wantCode: "112233",
			wantRule: "Generic numeric code",
		},
		{
			name: "Microsoft security code",
			email: "From: account-security@microsoft.com\r\n" +
				"Subject: Microsoft security code\r\n" +
				"Content-Type: text/plain\r\n" +
				"\r\n" +
				"Your security code: 7654321\r\n",
			wantCode: "7654321",
			wantRule: "Microsoft",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseEmail(strings.NewReader(tt.email))
			if err != nil {
				t.Fatalf("ParseEmail: %v", err)
			}

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil result, got code=%q rule=%q", result.Code, result.RuleName)
				}
				return
			}

			if result == nil {
				t.Fatal("expected result, got nil")
			}
			if result.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", result.Code, tt.wantCode)
			}
			if result.RuleName != tt.wantRule {
				t.Errorf("RuleName = %q, want %q", result.RuleName, tt.wantRule)
			}
			if result.BodyHash == "" {
				t.Error("BodyHash should not be empty")
			}
		})
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<p>Hello</p>", " Hello "},
		{"no tags", "no tags"},
		{"<b>bold</b> and <i>italic</i>", " bold  and  italic "},
		{"<div><p>nested</p></div>", "  nested  "},
	}

	for _, tt := range tests {
		got := stripHTML(tt.input)
		if got != tt.want {
			t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
