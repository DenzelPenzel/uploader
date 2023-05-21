package fake_auth

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type FakeAuth struct{}

func (ma FakeAuth) StartSession(c *gin.Context) {}

func (ma FakeAuth) ClearSession(w http.ResponseWriter) {}

func (ma FakeAuth) Authenticate(r *http.Request) bool {
	return true
}
