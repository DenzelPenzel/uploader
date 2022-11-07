package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"golang.org/x/crypto/pbkdf2"
	"net/http"
	"time"
)

const authCookie = "authSecret"

type (
	secret []byte

	Authorizer struct {
		secret secret
	}
)

func New(sharedSecret string) (Authorizer, error) {
	ss, err := parseSecret([]byte(sharedSecret))
	if err != nil {
		return Authorizer{}, err
	}
	return Authorizer{
		secret: ss,
	}, nil
}

func (a *Authorizer) StartSession(write http.ResponseWriter, request *http.Request) {
	s, err := getRequestSecret(request)
	if err != nil {
		http.Error(write, "Invalid secret", http.StatusBadRequest)
		return
	}

	if !isSecretsEqual(s, a.secret) {
		http.Error(write, "Incorrect secret", http.StatusUnauthorized)
		return
	}

	a.createCookie(write)
}

func (a *Authorizer) ClearSession(write http.ResponseWriter) {
	http.SetCookie(write, &http.Cookie{
		Name:     authCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
	})
}

func (a *Authorizer) Authenticate(request *http.Request) bool {
	cookie, err := request.Cookie(authCookie)
	if err != nil {
		return false
	}
	s, err := getSecretFromBase64(cookie.Value)
	if err != nil {
		return false
	}

	return isSecretsEqual(s, a.secret)
}

func (a *Authorizer) createCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookie,
		Value:    base64.StdEncoding.EncodeToString(a.secret),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(time.Hour * 24 * 30),
	})
}

type SecretRequest struct {
	SecretKey string `json:"secretKey"`
}

func getRequestSecret(r *http.Request) (secret, error) {
	var req SecretRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return secret{}, err
	}
	return parseSecret([]byte(req.SecretKey))
}

func getSecretFromBase64(b64encoded string) (secret, error) {
	if len(b64encoded) == 0 {
		return secret{}, errors.New("invalid secret")
	}

	decoded, err := base64.StdEncoding.DecodeString(b64encoded)
	if err != nil {
		return secret{}, err
	}

	return secret(decoded), nil
}

func parseSecret(key []byte) (secret, error) {
	if len(key) == 0 {
		return secret{}, errors.New("secret key in empty")
	}

	hash := pbkdf2.Key(key, []byte{1, 2, 3, 4}, 1000, 32, sha256.New)

	return secret(hash), nil
}

func isSecretsEqual(a, b secret) bool {
	return subtle.ConstantTimeCompare(a, b) != 0
}
