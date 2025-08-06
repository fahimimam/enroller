package utils

import (
	emailverifier "github.com/AfterShip/email-verifier"
	"math/rand"
	"regexp"
)

type UserType string

const (
	Employer UserType = "employer"
	Employee UserType = "employee"
)

func IsValueEmpty(val string) bool {
	if val == "" {
		return true
	}
	return false
}

func GenerateCode(low, hi int) int {
	return low + rand.Intn(hi-low)
}

func IsInValidEmail(email string) bool {
	verifier := emailverifier.NewVerifier()
	ret, err := verifier.Verify(email)
	if err != nil {
		return true
	}
	if !ret.Syntax.Valid {
		return true
	}
	return false
}

func IsValidPhoneNumber(phoneNumber string) bool {
	regex := "^(?:\\+?88|0088)?01[15-9]\\d{8}$"
	reg := regexp.MustCompile(regex)
	if !reg.MatchString(phoneNumber) {
		return false
	}
	return true
}

func IsUserType(userType UserType) bool {
	if userType == Employer || userType == Employee {
		return true
	}
	return false
}
