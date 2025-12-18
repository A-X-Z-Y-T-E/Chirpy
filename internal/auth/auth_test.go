package auth

import (
	"testing"

	"github.com/google/uuid"
)

func TestHashAndCheckPassword(t *testing.T) {
	pw := "supersecret123"

	hash, err := HashPassword(pw)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if hash == "" {
		t.Fatalf("expected non-empty hash")
	}

	ok, err := CheckPasswordHash(pw, hash)
	if err != nil {
		t.Fatalf("CheckPasswordHash error: %v", err)
	}
	if !ok {
		t.Fatalf("expected password to verify")
	}

	// Negative case
	ok, err = CheckPasswordHash("wrongpass", hash)
	if err != nil {
		t.Fatalf("CheckPasswordHash error (negative case): %v", err)
	}
	if ok {
		t.Fatalf("expected wrong password to fail verification")
	}
}

func TestJWT(t *testing.T) {
	tokenSecret := "my-super-secret-key"
	userID := uuid.New()

	// Test MakeJWT
	token, err := MakeJWT(userID, tokenSecret)
	if err != nil {
		t.Fatalf("MakeJWT error: %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty token")
	}

	// Test ValidateJWT with valid token
	parsedUserID, err := ValidateJWT(token, tokenSecret)
	if err != nil {
		t.Fatalf("ValidateJWT error: %v", err)
	}
	if parsedUserID != userID {
		t.Fatalf("expected user ID %v, got %v", userID, parsedUserID)
	}

	// Test with wrong secret
	_, err = ValidateJWT(token, "wrong-secret")
	if err == nil {
		t.Fatalf("expected error with wrong secret")
	}

	// Test with expired token
	expiredToken, err := MakeJWT(userID, tokenSecret)
	if err != nil {
		t.Fatalf("MakeJWT error for expired token: %v", err)
	}
	_, err = ValidateJWT(expiredToken, tokenSecret)
	if err == nil {
		t.Fatalf("expected error with expired token")
	}

	// Test with malformed token
	_, err = ValidateJWT("not.a.valid.token", tokenSecret)
	if err == nil {
		t.Fatalf("expected error with malformed token")
	}
}

func TestBearerToken(t *testing.T) {
	// Test valid bearer token
	headers := make(map[string][]string)
	headers["Authorization"] = []string{"Bearer my-secret-token-123"}

	token, err := GetBearerToken(headers)
	if err != nil {
		t.Fatalf("GetBearerToken error: %v", err)
	}
	if token != "my-secret-token-123" {
		t.Fatalf("expected token 'my-secret-token-123', got '%s'", token)
	}

	// Test missing authorization header
	emptyHeaders := make(map[string][]string)
	_, err = GetBearerToken(emptyHeaders)
	if err == nil {
		t.Fatalf("expected error with missing authorization header")
	}

	// Test malformed authorization header (no Bearer prefix)
	malformedHeaders := make(map[string][]string)
	malformedHeaders["Authorization"] = []string{"my-secret-token-123"}
	_, err = GetBearerToken(malformedHeaders)
	if err == nil {
		t.Fatalf("expected error with malformed authorization header")
	}

	// Test wrong prefix
	wrongPrefixHeaders := make(map[string][]string)
	wrongPrefixHeaders["Authorization"] = []string{"Basic my-secret-token-123"}
	_, err = GetBearerToken(wrongPrefixHeaders)
	if err == nil {
		t.Fatalf("expected error with wrong prefix")
	}

	// Test empty token
	emptyTokenHeaders := make(map[string][]string)
	emptyTokenHeaders["Authorization"] = []string{"Bearer "}
	token, err = GetBearerToken(emptyTokenHeaders)
	if err != nil {
		t.Fatalf("GetBearerToken error with empty token: %v", err)
	}
	if token != "" {
		t.Fatalf("expected empty token, got '%s'", token)
	}

	// Test too many parts
	tooManyPartsHeaders := make(map[string][]string)
	tooManyPartsHeaders["Authorization"] = []string{"Bearer token extra-part"}
	_, err = GetBearerToken(tooManyPartsHeaders)
	if err == nil {
		t.Fatalf("expected error with too many parts")
	}
}
