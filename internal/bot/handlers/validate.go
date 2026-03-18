package handlers

import (
	"errors"
	"regexp"
	"strings"

	"expense-bot/internal/domain"

	"github.com/shopspring/decimal"
)

var amountRegex = regexp.MustCompile(`^\d{1,12}(\.\d{1,6})?$`)

// TRC20/USDT address: starts with T, 34 chars, base58
var trc20Regex = regexp.MustCompile(`^T[1-9A-HJ-NP-Za-km-z]{33}$`)

// Card number: 13-19 digits (optionally with spaces/dashes)
var cardRegex = regexp.MustCompile(`^[\d\s\-]{13,23}$`)

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
	if len(input) == 0 {
		return "", errors.New("комментарий не может быть пустым")
	}
	if len(input) > 500 {
		return "", errors.New("комментарий не может быть длиннее 500 символов")
	}
	return input, nil
}

// ValidateAddress validates wallet address or card number based on payment method.
func ValidateAddress(input string) (string, error) {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return "", errors.New("адрес не может быть пустым")
	}
	if len(input) > 256 {
		return "", errors.New("адрес не может быть длиннее 256 символов")
	}
	return input, nil
}

// ValidateCryptoAddress validates USDT/TRX wallet address (TRC20 format).
func ValidateCryptoAddress(input string) (string, error) {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return "", errors.New("адрес кошелька не может быть пустым")
	}
	if !trc20Regex.MatchString(input) {
		return "", errors.New("некорректный адрес TRC20-кошелька. Адрес должен начинаться с T и содержать 34 символа")
	}
	return input, nil
}

// ValidateCardNumber validates card number format.
func ValidateCardNumber(input string) (string, error) {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return "", errors.New("реквизиты не могут быть пустыми")
	}
	// Allow free-form card details (number, IBAN, etc.)
	if len(input) > 256 {
		return "", errors.New("реквизиты не могут быть длиннее 256 символов")
	}
	return input, nil
}

func ValidateAccount(input string) (string, error) {
	if len(input) == 0 {
		return "", errors.New("аккаунт не может быть пустым")
	}
	if len(input) > 256 {
		return "", errors.New("аккаунт не может быть длиннее 256 символов")
	}
	return input, nil
}

// ChooseAddressValidator returns the right validator based on payment method.
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
