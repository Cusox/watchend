package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const tokenBytes = 32

var ErrInvalidPassword = errors.New("invalid password")

func GenerateHashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password must not be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	return string(hash), err
}

func VerifyHashPassword(hash, password string) error {
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return ErrInvalidPassword
	}
	return nil
}
