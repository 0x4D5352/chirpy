package auth

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestJWT(t *testing.T) {
	id := uuid.New()
	password := "chirpy"
	duration, err := time.ParseDuration("10s")
	if err != nil {
		t.Errorf("Error when parsing duration: %s", err)
	}
	token, err := MakeJWT(id, password, duration)
	if err != nil {
		t.Errorf("Error when creating JWT: %s", err)
	}
	validatedID, err := ValidateJWT(token, password)
	if err != nil {
		t.Errorf("Error when validating JWT: %s", err)
	}
	if validatedID != id {
		t.Errorf("Valided ID %v did not match starting ID %v", validatedID, id)
	}
	wrongID, err := ValidateJWT(token, "wrongpassword")
	if err == nil {
		t.Errorf("No error when using wrong password, got id %v", wrongID)
	}
	time.Sleep(duration)
	expiredToken, err := ValidateJWT(token, password)
	if err == nil {
		t.Errorf("No error when using expired token, got token %v", expiredToken)
	}
}

func TestBearerToken(t *testing.T) {
	req, err := http.NewRequest("GET", "https://www.example.com", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer your_token")
	req.Header.Set("User-Agent", "My-Go-App")

	bearer, err := GetBearerToken(req.Header)
	if err != nil {
		t.Errorf("Error when trying to pull bearer token %s", err)
	}

	if bearer != "your_token" {
		t.Errorf("Extracted token %s does not match input token your_token", bearer)
	}

	bad_req, err := http.NewRequest("GET", "https://www.example.com", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	bad_req.Header.Add("Content-Type", "application/json")
	bearer, err = GetBearerToken(bad_req.Header)
	if err == nil {
		t.Errorf("No Error when trying to pull bearer token")
	}
}
