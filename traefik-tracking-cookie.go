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
	length              int
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return &TraefikTrackingCookie{
		next:                next,
		name:                name,
		cookieDomain:        config.Domain,
		clientCookieName:    fmt.Sprintf("%s-%s", config.CookieNamePrefix, config.ClientCookieName),
		sessionCookieName:   fmt.Sprintf("%s-%s", config.CookieNamePrefix, config.SessionCookieName),
		clientCookieExpires: config.ClientCookieExpires,
		cookieHttpOnly:      config.HttpOnly,
		cookieSecure:        config.Secure,
		length:              config.Length,
	}, nil
}

func (a *TraefikTrackingCookie) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	clientCookie, err := req.Cookie(a.clientCookieName)

	if err != nil || time.Now().Add(time.Duration(float64(a.clientCookieExpires)*0.75)*time.Second).After(clientCookie.Expires) {
		if val, err := generateRandomString(a.length); err == nil {
			var expires time.Time

			if a.clientCookieExpires > 0 {
				expires = time.Now().Add(time.Duration(a.clientCookieExpires) * time.Second)
			}

			clientCookie = &http.Cookie{
				Name:     a.clientCookieName,
				Value:    val,
				Domain:   a.cookieDomain,
				Path:     "/",
				MaxAge:   a.clientCookieExpires,
				Expires:  expires,
				HttpOnly: a.cookieHttpOnly,
				Secure:   a.cookieSecure,
			}

			http.SetCookie(rw, clientCookie)
		}
	}

	sessionCookie, err := req.Cookie(a.sessionCookieName)

	if err != nil {
		if val, err := generateRandomString(a.length); err == nil {
			sessionCookie = &http.Cookie{
				Name:     a.sessionCookieName,
				Value:    val,
				Domain:   a.cookieDomain,
				Path:     "/",
				MaxAge:   0,
				HttpOnly: a.cookieHttpOnly,
				Secure:   a.cookieSecure,
			}

			http.SetCookie(rw, sessionCookie)
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
