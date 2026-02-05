package auth

import "strings"

// NormalizePhone приводит номер к E.164-подобному виду: + и только цифры.
func NormalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}
	var b strings.Builder
	if len(phone) > 0 && phone[0] == '+' {
		b.WriteByte('+')
		phone = phone[1:]
	}
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
