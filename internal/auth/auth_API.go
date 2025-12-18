package auth

import (
	"errors"
	"net/http"
	"strings"
)

func GetAPIKEY(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("no authorization header")
	}

	// Split "ApiKey <key>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "ApiKey" {
		return "", errors.New("malformed authorization header")
	}
	return parts[1], nil
}
