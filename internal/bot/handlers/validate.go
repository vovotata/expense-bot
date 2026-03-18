package handlers

import (
	"errors"
	"regexp"

	"github.com/shopspring/decimal"
)

var amountRegex = regexp.MustCompile(`^\d{1,12}(\.\d{1,6})?$`)

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

func ValidateAddress(input string) (string, error) {
	if len(input) == 0 {
		return "", errors.New("адрес не может быть пустым")
	}
	if len(input) > 256 {
		return "", errors.New("адрес не может быть длиннее 256 символов")
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
