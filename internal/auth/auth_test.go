package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	password := "chirpy"
	hash, err := HashPassword(password)
	if err != nil {
		t.Errorf("Error generating password: %s", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		t.Errorf("%v should hash %s correctly", hash, password)
	}
	notPass := "somethingelse"
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(notPass))
	if err != bcrypt.ErrMismatchedHashAndPassword {
		t.Errorf("%v and %s should be mismatched", hash, notPass)
	}
}

func TestCheckPassword(t *testing.T) {
	password := []byte("chirpy")
	hash, err := bcrypt.GenerateFromPassword(password, 10)
	if err != nil {
		t.Errorf("Error generating password: %s", err)
	}
	if CheckPasswordHash(string(password), string(hash)) != nil {
		t.Errorf("%v should match %s", hash, password)
	}
	notPass := "somethingelse"
	err = CheckPasswordHash(notPass, string(hash))
	if err != bcrypt.ErrMismatchedHashAndPassword {
		t.Errorf("%v and %s should be mismatched", hash, notPass)
	}
}
