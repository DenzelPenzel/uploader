package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
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

	Login struct {
		Secret string `form:"secretKey" json:"secretKey" xml:"secretKey" binding:"required"`
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

func (a *Authorizer) StartSession(c *gin.Context) {
	var json Login

	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s, err := parseSecret([]byte(json.Secret))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid secret"})
		return
	}

	if !isSecretsEqual(s, a.secret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect secret"})
		return
	}

	cookie := &http.Cookie{
		Name:     authCookie,
		Value:    base64.StdEncoding.EncodeToString(a.secret),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(time.Hour * 24 * 30),
	}
	http.SetCookie(c.Writer, cookie)
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

func (a *Authorizer) Authenticate(c *http.Request) bool {
	cookie, err := c.Cookie(authCookie)
	if err != nil {
		return false
	}
	s, err := getSecretFromBase64(cookie.Value)
	if err != nil {
		return false
	}

	return isSecretsEqual(s, a.secret)
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
