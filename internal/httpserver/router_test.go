package httpserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

const (
	testUser = "admin"
	testPass = "correct-horse-battery-staple"
)

func marker(body string) fiber.Handler {
	return func(c *fiber.Ctx) error { return c.SendString(body) }
}

// protectedStub registers several routes and does NOT opt out: every one of
// them must be protected, including nested and parameterised routes.
type protectedStub struct{}

func (protectedStub) RegisterRoutes(r fiber.Router) {
	g := r.Group("/things")
	g.Get("/", marker("protected"))
	g.Post("/", marker("protected"))
	g.Get("/:id", marker("protected"))
	g.Get("/:id/history", marker("protected"))
}

// publicStub opts its prefix out, like the TCOP webhook handler does.
type publicStub struct{}

func (publicStub) RegisterRoutes(r fiber.Router) {
	r.Post("/hooks/:token", marker("public"))
}

func (publicStub) PublicPathPrefixes() []string {
	return []string{"/hooks/"}
}

func newTestApp() *fiber.App {
	app := fiber.New()
	NewRouter(app, AdminCredentials{Username: testUser, Password: testPass}, protectedStub{}, publicStub{})
	return app
}

type result struct {
	status int
	body   string
	header http.Header
}

func do(t *testing.T, app *fiber.App, method, target, user, pass string) result {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	if user != "" || pass != "" {
		req.SetBasicAuth(user, pass)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return result{status: resp.StatusCode, body: string(b), header: resp.Header}
}

func TestProtectedRoutes(t *testing.T) {
	app := newTestApp()
	routes := []struct{ method, target string }{
		{"GET", "/api/v1/things/"},
		{"POST", "/api/v1/things/"},
		{"GET", "/api/v1/things/42"},
		{"GET", "/api/v1/things/42/history"},
	}
	for _, rt := range routes {
		if got := do(t, app, rt.method, rt.target, "", ""); got.status != 401 {
			t.Errorf("%s %s without credentials: status %d, want 401", rt.method, rt.target, got.status)
		}
		if got := do(t, app, rt.method, rt.target, testUser, "wrong-password-value"); got.status != 401 {
			t.Errorf("%s %s wrong password: status %d, want 401", rt.method, rt.target, got.status)
		}
		if got := do(t, app, rt.method, rt.target, "wrong-user", testPass); got.status != 401 {
			t.Errorf("%s %s wrong username: status %d, want 401", rt.method, rt.target, got.status)
		}
		if got := do(t, app, rt.method, rt.target, testUser, testPass); got.status != 200 || got.body != "protected" {
			t.Errorf("%s %s valid credentials: status %d body %q, want 200 protected", rt.method, rt.target, got.status, got.body)
		}
	}
}

func TestUnauthorizedAdvertisesBasicRealm(t *testing.T) {
	got := do(t, newTestApp(), "GET", "/api/v1/things/", "", "")
	if h := got.header.Get("WWW-Authenticate"); h == "" {
		t.Errorf("401 response has no WWW-Authenticate header; clients and browsers cannot prompt for credentials")
	}
}

func TestPublicPrefixNeedsNoAdminCredentials(t *testing.T) {
	got := do(t, newTestApp(), "POST", "/api/v1/hooks/some-token", "", "")
	if got.status != 200 || got.body != "public" {
		t.Errorf("public route without credentials: status %d body %q, want 200 public", got.status, got.body)
	}
}

func TestHealthzIsPublic(t *testing.T) {
	got := do(t, newTestApp(), "GET", "/healthz", "", "")
	if got.status != 200 || got.body != "ok" {
		t.Errorf("healthz: status %d body %q, want 200 ok", got.status, got.body)
	}
}

// TestPublicPrefixDoesNotLeak sends path-manipulation probes without any
// credentials. None of them may ever reach a protected handler.
func TestPublicPrefixDoesNotLeak(t *testing.T) {
	app := newTestApp()
	probes := []string{
		"/api/v1/hooks/../things/",
		"/api/v1/hooks/x/../../things/",
		"/api/v1/hooks/%2e%2e/things/",
		"/api/v1/hooks/..%2fthings/",
		"/api/v1/hooks/x%2f..%2f..%2fthings/",
		"/api/v1/hooks/%252e%252e/things/",
		"/api/v1/hooks/..;/things/",
		"//api/v1/things/",
		"/api/v1//things/",
		"/api/v1/./things/",
		"/./api/v1/things/",
		"/API/V1/THINGS/",
		"/api/v1/Things/",
		"/api/v1/hooks/x/./../../things/",
		"/healthz/../api/v1/things/",
		"/api/v1/hooksX/things/",
		"/api/v1/hooks",
	}
	for _, p := range probes {
		got := do(t, app, "GET", p, "", "")
		if got.body == "protected" {
			t.Errorf("BYPASS: GET %s reached a protected handler without credentials (status %d)", p, got.status)
		}
	}
}

// TestGroupMiddlewareIsOrderDependent documents WHY NewRouter attaches auth
// to the group before registering routes instead of putting middleware on a
// second group with the same prefix: Fiber turns group middleware into a
// prefix-wide Use, so whether the public route survives depends only on
// which group was registered first.
func TestGroupMiddlewareIsOrderDependent(t *testing.T) {
	block := func(c *fiber.Ctx) error { return c.SendStatus(401) }

	publicFirst := fiber.New()
	publicFirst.Group("/api/v1").Post("/hooks/:token", marker("public"))
	publicFirst.Group("/api/v1", block).Get("/things", marker("protected"))

	protectedFirst := fiber.New()
	protectedFirst.Group("/api/v1", block).Get("/things", marker("protected"))
	protectedFirst.Group("/api/v1").Post("/hooks/:token", marker("public"))

	a := do(t, publicFirst, "POST", "/api/v1/hooks/x", "", "")
	b := do(t, protectedFirst, "POST", "/api/v1/hooks/x", "", "")
	t.Logf("public route registered first  -> status %d", a.status)
	t.Logf("public route registered second -> status %d", b.status)
	if a.status == b.status {
		t.Errorf("expected registration order to change the outcome (the pitfall this design avoids); both gave %d", a.status)
	}
}
