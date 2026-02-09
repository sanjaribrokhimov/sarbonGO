package util

import (
	"fmt"
	"strings"
	"unicode"
)

// NormalizeE164 does a strict-ish normalization to E.164: +<digits>.
// It rejects empty values and values that don't contain enough digits.
func NormalizeE164(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("phone is required")
	}

	var digits []rune
	for _, r := range s {
		if r == '+' {
			continue
		}
		if unicode.IsDigit(r) {
			digits = append(digits, r)
			continue
		}
		// ignore separators commonly used by users
		switch r {
		case ' ', '-', '(', ')':
			continue
		default:
			return "", fmt.Errorf("phone contains invalid characters")
		}
	}

	if len(digits) < 8 || len(digits) > 15 {
		return "", fmt.Errorf("phone must be in E.164 format")
	}
	return "+" + string(digits), nil
}

