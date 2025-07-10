package traefik_tracking_cookie

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

const cookieNamePrefix = "ttc"
const clientCookieName = "cid"
const sessionCookieName = "sid"

type Config struct {
	Domain              string `json:"domain,omitempty" yaml:"domain,omitempty" toml:"domain,omitempty"`
	CookieNamePrefix    string `json:"cookienameprefix,omitempty" yaml:"cookienameprefix,omitempty" toml:"cookienameprefix,omitempty"`
	ClientCookieName    string `json:"clientcookiename,omitempty" yaml:"clientcookiename,omitempty" toml:"clientcookiename,omitempty"`
	SessionCookieName   string `json:"sessioncookiename,omitempty" yaml:"sessioncookiename,omitempty" toml:"sessioncookiename,omitempty"`
	ClientCookieExpires int    `json:"clientcookieexpires,omitempty" yaml:"clientcookieexpires,omitempty" toml:"clientcookieexpires,omitempty"`
	HttpOnly            bool   `json:"httponly,omitempty" yaml:"httponly,omitempty" toml:"httponly,omitempty"`
	Secure              bool   `json:"secure,omitempty" yaml:"secure,omitempty" toml:"secure,omitempty"`
	Length              int    `json:"length,omitempty" yaml:"length,omitempty" toml:"length,omitempty"`
	SameSite            string `json:"samesite,omitempty" yaml:"samesite,omitempty" toml:"samesite,omitempty"` // "strict", "lax", "none", or ""
}

func (c *Config) Validate() error {
	if c.Length < 8 || c.Length > 128 {
		return fmt.Errorf("length must be between 8 and 128, got %d", c.Length)
	}
	if c.ClientCookieExpires < 0 {
		return fmt.Errorf("clientCookieExpires cannot be negative, got %d", c.ClientCookieExpires)
	}
	if c.SameSite != "" && c.SameSite != "strict" && c.SameSite != "lax" && c.SameSite != "none" {
		return fmt.Errorf("samesite must be 'strict', 'lax', 'none', or empty, got %s", c.SameSite)
	}
	return nil
}

func (c *Config) getSameSite() http.SameSite {
	switch c.SameSite {
	case "strict":
		return http.SameSiteStrictMode
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteDefaultMode
	}
}

func CreateConfig() *Config {
	return &Config{
		CookieNamePrefix:    cookieNamePrefix,
		ClientCookieName:    clientCookieName,
		SessionCookieName:   sessionCookieName,
		ClientCookieExpires: 365 * 24 * 60 * 60,
		Length:              32,
	}
}

type TraefikTrackingCookie struct {
	next                http.Handler
	name                string
	cookieDomain        string
	clientCookieName    string
	sessionCookieName   string
	clientCookieExpires int
	cookieHttpOnly      bool
	cookieSecure        bool
	cookieSameSite      http.SameSite
	length              int
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if next == nil {
		return nil, fmt.Errorf("next handler cannot be nil")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &TraefikTrackingCookie{
		next:                next,
		name:                name,
		cookieDomain:        config.Domain,
		clientCookieName:    fmt.Sprintf("%s-%s", config.CookieNamePrefix, config.ClientCookieName),
		sessionCookieName:   fmt.Sprintf("%s-%s", config.CookieNamePrefix, config.SessionCookieName),
		clientCookieExpires: config.ClientCookieExpires,
		cookieHttpOnly:      config.HttpOnly,
		cookieSecure:        config.Secure,
		cookieSameSite:      config.getSameSite(),
		length:              config.Length,
	}, nil
}

func (a *TraefikTrackingCookie) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	_, err := req.Cookie(a.clientCookieName)

	// Only generate client cookie if it doesn't exist
	// Note: We can't check expiration from the request cookie since browsers
	// don't send expires information back to the server
	if err != nil {
		val, err := generateRandomString(a.length)
		if err != nil {
			rw.Header().Set("X-Client-Cookie-Error", "failed to generate client cookie")
		} else {
			var expires time.Time

			if a.clientCookieExpires > 0 {
				expires = time.Now().Add(time.Duration(a.clientCookieExpires) * time.Second)
			}

			clientCookie := &http.Cookie{
				Name:     a.clientCookieName,
				Value:    val,
				Domain:   a.cookieDomain,
				Path:     "/",
				MaxAge:   a.clientCookieExpires,
				Expires:  expires,
				HttpOnly: a.cookieHttpOnly,
				Secure:   a.cookieSecure,
				SameSite: a.cookieSameSite,
			}

			http.SetCookie(rw, clientCookie)
		}
	}

	_, err = req.Cookie(a.sessionCookieName)
	if err != nil {
		val, err := generateRandomString(a.length)
		if err != nil {
			rw.Header().Set("X-Session-Cookie-Error", "failed to generate session cookie")
		} else {
			sessionCookie := &http.Cookie{
				Name:     a.sessionCookieName,
				Value:    val,
				Domain:   a.cookieDomain,
				Path:     "/",
				MaxAge:   0, // Session cookie
				HttpOnly: a.cookieHttpOnly,
				Secure:   a.cookieSecure,
				SameSite: a.cookieSameSite,
			}

			http.SetCookie(rw, sessionCookie)
		}
	}

	a.next.ServeHTTP(rw, req)
}

func generateRandomString(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be positive, got %d", length)
	}

	// Calculate buffer size needed for base64 encoding
	// base64 encoding produces 4 characters for every 3 bytes
	bufferSize := (length*3 + 3) / 4 // Round up division
	buffer := make([]byte, bufferSize)

	_, err := rand.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	encoded := base64.URLEncoding.EncodeToString(buffer)
	if len(encoded) > length {
		encoded = encoded[:length]
	}

	return encoded, nil
}
