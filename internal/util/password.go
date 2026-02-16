package util

import (
	"fmt"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

func ValidatePassword(pw string) error {
	if len(pw) < 6 {
		return fmt.Errorf("password must be at least 6 characters")
	}
	hasLetter := false
	for _, r := range pw {
		if unicode.IsLetter(r) {
			hasLetter = true
			break
		}
	}
	if !hasLetter {
		return fmt.Errorf("password must contain at least 1 letter")
	}
	return nil
}

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ComparePassword(hash string, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}
