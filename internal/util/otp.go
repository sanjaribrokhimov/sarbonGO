package util

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"unicode"
)

func GenerateNumericOTP(length int) (string, error) {
	if length < 4 || length > 8 {
		return "", fmt.Errorf("otp length must be 4..8")
	}
	var b strings.Builder
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteByte(byte('0' + n.Int64()))
	}
	return b.String(), nil
}

func IsNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

