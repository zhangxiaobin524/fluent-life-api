package validator

import (
	"regexp"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

func IsEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func IsPhone(phone string) bool {
	return phoneRegex.MatchString(phone)
}

func IsEmailOrPhone(identifier string) bool {
	return IsEmail(identifier) || IsPhone(identifier)
}

func ValidatePassword(password string) bool {
	return len(password) >= 6
}






