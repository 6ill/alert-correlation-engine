package httpserver

import (
	"crypto/sha256"
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
)

type AdminCredentials struct {
	Username string
	Password string
}

func adminAuth(admin AdminCredentials, publicPrefixes []string) fiber.Handler {
	wantUser := sha256.Sum256([]byte(admin.Username))
	wantPass := sha256.Sum256([]byte(admin.Password))

	return basicauth.New(basicauth.Config{
		Realm: "Alert Correlation Engine",
		// Next is evaluated against c.Path(), the same normalized path Fiber
		// routes on, so this check and the router cannot disagree about
		// which route a request is for.
		Next: func(c *fiber.Ctx) bool {
			path := c.Path()
			for _, prefix := range publicPrefixes {
				if strings.HasPrefix(path, prefix) {
					return true
				}
			}
			return false
		},
		// Digests are compared instead of the raw strings so the comparison
		// is constant-time regardless of input length, and both comparisons
		// always run so a wrong username is not distinguishable by timing.
		Authorizer: func(user, pass string) bool {
			gotUser := sha256.Sum256([]byte(user))
			gotPass := sha256.Sum256([]byte(pass))
			userOK := subtle.ConstantTimeCompare(gotUser[:], wantUser[:])
			passOK := subtle.ConstantTimeCompare(gotPass[:], wantPass[:])
			return userOK&passOK == 1
		},
	})
}
