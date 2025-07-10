package traefik_tracking_cookie_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	traefik_tracking_cookie "github.com/stabelo/traefik-tracking-cookie"
)

func TestTraefikTrackingCookie(t *testing.T) {
	tests := []struct {
		name   string
		config *traefik_tracking_cookie.Config
		want   int // expected number of Set-Cookie headers
	}{
		{
			name: "basic functionality",
			config: &traefik_tracking_cookie.Config{
				Domain:   "example.com",
				HttpOnly: true,
				Secure:   true,
				Length:   16,
				SameSite: "lax",
			},
			want: 2,
		},
		{
			name: "with custom names",
			config: &traefik_tracking_cookie.Config{
				Domain:              "test.com",
				CookieNamePrefix:    "custom",
				ClientCookieName:    "client",
				SessionCookieName:   "session",
				ClientCookieExpires: 3600,
				HttpOnly:            false,
				Secure:              false,
				Length:              32,
				SameSite:            "strict",
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

			handler, err := traefik_tracking_cookie.New(ctx, next, tt.config, "test-plugin")
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			recorder := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
			if err != nil {
				t.Fatal(err)
			}

			handler.ServeHTTP(recorder, req)

			cookies := recorder.Header().Values("Set-Cookie")
			if len(cookies) != tt.want {
				t.Errorf("Expected %d Set-Cookie headers, got %d", tt.want, len(cookies))
			}

			// Verify SameSite attribute is set correctly
			if tt.config.SameSite != "" {
				found := false
				for _, cookie := range cookies {
					if strings.Contains(cookie, "SameSite="+strings.Title(tt.config.SameSite)) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("SameSite attribute not found in cookies")
				}
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *traefik_tracking_cookie.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &traefik_tracking_cookie.Config{
				Length:              32,
				ClientCookieExpires: 3600,
				SameSite:            "lax",
			},
			wantErr: false,
		},
		{
			name: "length too small",
			config: &traefik_tracking_cookie.Config{
				Length:              5,
				ClientCookieExpires: 3600,
			},
			wantErr: true,
		},
		{
			name: "length too large",
			config: &traefik_tracking_cookie.Config{
				Length:              200,
				ClientCookieExpires: 3600,
			},
			wantErr: true,
		},
		{
			name: "negative expires",
			config: &traefik_tracking_cookie.Config{
				Length:              32,
				ClientCookieExpires: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid samesite",
			config: &traefik_tracking_cookie.Config{
				Length:   32,
				SameSite: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidation(t *testing.T) {
	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	t.Run("nil config", func(t *testing.T) {
		_, err := traefik_tracking_cookie.New(ctx, next, nil, "test")
		if err == nil {
			t.Error("Expected error for nil config")
		}
	})

	t.Run("nil handler", func(t *testing.T) {
		config := traefik_tracking_cookie.CreateConfig()
		_, err := traefik_tracking_cookie.New(ctx, nil, config, "test")
		if err == nil {
			t.Error("Expected error for nil handler")
		}
	})
}

func TestCookieExpiration(t *testing.T) {
	config := traefik_tracking_cookie.CreateConfig()
	config.ClientCookieExpires = 1 // 1 second

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := traefik_tracking_cookie.New(ctx, next, config, "test")
	if err != nil {
		t.Fatal(err)
	}

	// First request - should set cookies
	recorder1 := httptest.NewRecorder()
	req1, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	handler.ServeHTTP(recorder1, req1)

	cookies1 := recorder1.Header().Values("Set-Cookie")
	if len(cookies1) != 2 {
		t.Fatalf("Expected 2 cookies on first request, got %d", len(cookies1))
	}

	// Extract the client cookie from the first response
	clientCookieValue := ""
	for _, cookieStr := range cookies1 {
		if strings.Contains(cookieStr, "ttc-cid=") {
			parts := strings.Split(cookieStr, "=")
			if len(parts) >= 2 {
				clientCookieValue = strings.Split(parts[1], ";")[0]
			}
		}
	}

	// Second request with existing cookie but not expired yet
	recorder2 := httptest.NewRecorder()
	req2, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)

	// Add the cookie from first request
	clientCookie := &http.Cookie{
		Name:    "ttc-cid",
		Value:   clientCookieValue,
		Expires: time.Now().Add(time.Hour), // Not expired
	}
	req2.AddCookie(clientCookie)

	handler.ServeHTTP(recorder2, req2)

	cookies2 := recorder2.Header().Values("Set-Cookie")
	// Should only set session cookie since client cookie exists and is not near expiration
	if len(cookies2) != 1 {
		t.Errorf("Expected 1 cookie on second request (session only), got %d", len(cookies2))
	}
}
