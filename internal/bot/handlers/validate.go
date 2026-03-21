package handlers

import (
	"errors"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"

	"expense-bot/internal/domain"

	"github.com/shopspring/decimal"
)

var amountRegex = regexp.MustCompile(`^\d{1,12}(\.\d{1,6})?$`)

// TRC20/USDT address: starts with T, 34 chars, base58
var trc20Regex = regexp.MustCompile(`^T[1-9A-HJ-NP-Za-km-z]{33}$`)

func ValidateAmount(input string) (string, error) {
	if !amountRegex.MatchString(input) {
		return "", errors.New("некорректный формат суммы, используйте: 12.452")
	}
	val, err := decimal.NewFromString(input)
	if err != nil {
		return "", errors.New("некорректный формат суммы")
	}
	if val.IsZero() || val.IsNegative() {
		return "", errors.New("сумма должна быть положительной")
	}
	return input, nil
}

func ValidateComment(input string) (string, error) {
	if utf8.RuneCountInString(input) == 0 {
		return "", errors.New("комментарий не может быть пустым")
	}
	if utf8.RuneCountInString(input) > 500 {
		return "", errors.New("комментарий не может быть длиннее 500 символов")
	}
	return input, nil
}

func ValidateAddress(input string) (string, error) {
	input = strings.TrimSpace(input)
	if utf8.RuneCountInString(input) == 0 {
		return "", errors.New("адрес не может быть пустым")
	}
	if utf8.RuneCountInString(input) > 256 {
		return "", errors.New("адрес не может быть длиннее 256 символов")
	}
	return input, nil
}

func ValidateCryptoAddress(input string) (string, error) {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return "", errors.New("адрес кошелька не может быть пустым")
	}
	if !trc20Regex.MatchString(input) {
		return "", errors.New("некорректный адрес кошелька. Адрес должен начинаться с T и содержать 34 символа (формат TRON)")
	}
	return input, nil
}

func ValidateCardNumber(input string) (string, error) {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return "", errors.New("реквизиты не могут быть пустыми")
	}

	// Extract only digits
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, input)

	// If it looks like a card number (13-19 digits), validate with Luhn
	if len(digits) >= 13 && len(digits) <= 19 {
		if !luhnCheck(digits) {
			return "", errors.New("некорректный номер карты. Проверьте и попробуйте снова")
		}
		// Format nicely: XXXX XXXX XXXX XXXX
		return formatCardNumber(digits), nil
	}

	// Otherwise accept as free-form requisites (IBAN, phone, etc.)
	if utf8.RuneCountInString(input) > 256 {
		return "", errors.New("реквизиты не могут быть длиннее 256 символов")
	}
	return input, nil
}

func luhnCheck(digits string) bool {
	sum := 0
	alt := false
	for i := len(digits) - 1; i >= 0; i-- {
		n := int(digits[i] - '0')
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}

func formatCardNumber(digits string) string {
	var parts []string
	for i := 0; i < len(digits); i += 4 {
		end := i + 4
		if end > len(digits) {
			end = len(digits)
		}
		parts = append(parts, digits[i:end])
	}
	return strings.Join(parts, " ")
}

func ValidateAccount(input string) (string, error) {
	if utf8.RuneCountInString(input) == 0 {
		return "", errors.New("аккаунт не может быть пустым")
	}
	if utf8.RuneCountInString(input) > 256 {
		return "", errors.New("аккаунт не может быть длиннее 256 символов")
	}
	return input, nil
}

func ChooseAddressValidator(pm domain.PaymentMethod) func(string) (string, error) {
	switch pm {
	case domain.PaymentUSDT, domain.PaymentTRX:
		return ValidateCryptoAddress
	case domain.PaymentCard:
		return ValidateCardNumber
	default:
		return ValidateAddress
	}
}

// EscapeHTML escapes user input for safe use in Telegram HTML messages.
func EscapeHTML(s string) string {
	return html.EscapeString(s)
}
