package traefik_tracking_cookie_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	traefik_tracking_cookie "github.com/stabelo/traefik-tracking-cookie"
)

func TestTraefikTrackingCookie(t *testing.T) {
	cfg := traefik_tracking_cookie.CreateConfig()
	cfg.Domain = "example.com"
	cfg.HttpOnly = true
	cfg.Secure = true

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := traefik_tracking_cookie.New(ctx, next, cfg, "traefik-tracking-cookie")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	cookies := recorder.Header().Values("Set-Cookie")

	if len(cookies) != 2 {
		t.Errorf("Set-Cookie header not set for both client and session")
	}
}
