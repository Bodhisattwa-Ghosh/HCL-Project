package validation

import (
	"errors"
	"net/mail"
	"strings"
	"unicode"
)

const (
	MaxQuestionLength = 280
	MaxOptionLength   = 100
	MinOptions        = 2
	MaxOptions        = 8
)

func CleanText(value string, max int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len([]rune(value)) > max {
		return ""
	}
	return value
}

func Email(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || len(value) > 254 {
		return "", errors.New("enter a valid email address")
	}
	return value, nil
}

func Password(value string) error {
	if len(value) < 10 || len(value) > 72 {
		return errors.New("password must be 10 to 72 characters")
	}
	var hasLetter, hasNumber bool
	for _, char := range value {
		hasLetter = hasLetter || unicode.IsLetter(char)
		hasNumber = hasNumber || unicode.IsNumber(char)
	}
	if !hasLetter || !hasNumber {
		return errors.New("password must include a letter and a number")
	}
	return nil
}

func Poll(question string, options []string) (string, []string, error) {
	question = CleanText(question, MaxQuestionLength)
	if len([]rune(question)) < 5 {
		return "", nil, errors.New("question must be 5 to 280 characters")
	}
	if len(options) < MinOptions || len(options) > MaxOptions {
		return "", nil, errors.New("a poll needs between 2 and 8 options")
	}
	seen := make(map[string]struct{}, len(options))
	cleaned := make([]string, 0, len(options))
	for _, option := range options {
		option = CleanText(option, MaxOptionLength)
		if len([]rune(option)) < 1 {
			return "", nil, errors.New("each option must be 1 to 100 characters")
		}
		key := strings.ToLower(option)
		if _, exists := seen[key]; exists {
			return "", nil, errors.New("options must be unique")
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, option)
	}
	return question, cleaned, nil
}
