package http

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"mime"
	nethttp "net/http"
	"net/url"
	"strings"
)

const csrfCookie = "watchend_csrf"

func csrfToken(r *nethttp.Request) string {
	c, _ := r.Cookie(csrfCookie)
	if c == nil {
		return ""
	}
	return c.Value
}

func (s *Server) newCSRF() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	raw := base64.RawURLEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(raw))

	return raw + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (s *Server) validCSRF(token string) bool {
	raw, sig, ok := strings.Cut(token, ".")
	if !ok {
		return false
	}
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(raw))

	return hmac.Equal(got, mac.Sum(nil))
}

func requestOrigin(r *nethttp.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]); forwarded == "http" || forwarded == "https" {
		scheme = forwarded
	}
	host := r.Host
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0]); forwarded != "" {
		host = forwarded
	}
	return scheme + "://" + host
}

func sameOrigin(r *nethttp.Request, origin string) bool {
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return false
	}
	return strings.EqualFold(parsed.Scheme+"://"+parsed.Host, requestOrigin(r))
}

func (s *Server) csrf(next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		cookie, err := r.Cookie(csrfCookie)
		if err != nil || !s.validCSRF(cookie.Value) {
			token, e := s.newCSRF()
			if e != nil {
				nethttp.Error(w, "internal server error", 500)
				return
			}

			cookie = &nethttp.Cookie{
				Name:     csrfCookie,
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				Secure:   s.secure,
				SameSite: nethttp.SameSiteStrictMode,
			}

			nethttp.SetCookie(w, cookie)

			r.AddCookie(cookie)
		}

		if r.Method != nethttp.MethodGet && r.Method != nethttp.MethodHead && r.Method != nethttp.MethodOptions {
			origin := r.Header.Get("Origin")
			token := r.Header.Get("X-CSRF-Token")
			if token == "" {
				mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
				var err error
				if mediaType == "multipart/form-data" {
					err = r.ParseMultipartForm(maxBodyBytes)
				} else {
					err = r.ParseForm()
				}
				if err != nil {
					nethttp.Error(w, "invalid request", 400)
					return
				}
				token = r.PostForm.Get("csrf_token")
			}

			if subtle.ConstantTimeCompare([]byte(token), []byte(cookie.Value)) != 1 {
				nethttp.Error(w, "invalid CSRF token", nethttp.StatusForbidden)
				return
			}
			if !sameOrigin(r, origin) && s.logger != nil {
				s.logger.Warn("request origin differs from server origin", "origin", origin, "request_origin", requestOrigin(r))
			}
		}
		next.ServeHTTP(w, r)
	})
}
