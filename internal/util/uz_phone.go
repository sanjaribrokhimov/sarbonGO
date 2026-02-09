package util

import (
	"fmt"
	"regexp"
	"strings"
)

var uzPhoneRe = regexp.MustCompile(`^\+998\d{9}$`)

// ValidateUzPhoneStrict enforces +998XXXXXXXXX (Uzbekistan) format.
func ValidateUzPhoneStrict(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if !uzPhoneRe.MatchString(s) {
		return "", fmt.Errorf("phone must match +998XXXXXXXXX")
	}
	return s, nil
}

