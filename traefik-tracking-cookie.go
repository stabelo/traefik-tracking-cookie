package traefik_tracking_cookie

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"
)

const defaultCookieName = "tid"

type Config struct {
	Domain   string `json:"domain,omitempty" yaml:"domain,omitempty" toml:"domain,omitempty"`
	Name     string `json:"name,omitempty" yaml:"name,omitempty" toml:"name,omitempty"`
	Expires  int    `json:"expires,omitempty" yaml:"expires,omitempty" toml:"expires,omitempty"`
	HttpOnly bool   `json:"httponly,omitempty" yaml:"httponly,omitempty" toml:"httponly,omitempty"`
	Secure   bool   `json:"secure,omitempty" yaml:"secure,omitempty" toml:"secure,omitempty"`
}

func CreateConfig() *Config {
	return &Config{
		Name:    defaultCookieName,
		Expires: 0,
	}
}

type TraefikTrackingCookie struct {
	next           http.Handler
	name           string
	cookieDomain   string
	cookieName     string
	cookieExpires  int
	cookieHttpOnly bool
	cookieSecure   bool
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return &TraefikTrackingCookie{
		next:           next,
		name:           name,
		cookieDomain:   config.Domain,
		cookieName:     config.Name,
		cookieExpires:  config.Expires,
		cookieHttpOnly: config.HttpOnly,
		cookieSecure:   config.Secure,
	}, nil
}

func (a *TraefikTrackingCookie) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	cookie, err := req.Cookie(a.cookieName)

	if err != nil {
		if val, err := generateRandomString(20); err == nil {
			var expires time.Time

			if a.cookieExpires > 0 {
				expires = time.Now().Add(time.Duration(a.cookieExpires) * time.Second)
			}

			cookie = &http.Cookie{
				Name:     a.cookieName,
				Value:    val,
				Domain:   a.cookieDomain,
				Path:     "/",
				MaxAge:   a.cookieExpires,
				Expires:  expires,
				HttpOnly: a.cookieHttpOnly,
				Secure:   a.cookieSecure,
			}

			http.SetCookie(rw, cookie)
		}
	}

	a.next.ServeHTTP(rw, req)
}

func generateRandomString(length int) (string, error) {
	buffer := make([]byte, length)
	_, err := rand.Read(buffer)

	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buffer)[:length], nil
}
