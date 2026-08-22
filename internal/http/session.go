package http

import (
	nethttp "net/http"
)

const sessionCookie = "watchend_session"

func (s *Server) setSession(w nethttp.ResponseWriter, token string) {
	nethttp.SetCookie(w, &nethttp.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(s.ttl.Seconds()),
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: nethttp.SameSiteLaxMode,
	})
}

func (s *Server) clearSession(w nethttp.ResponseWriter) {
	nethttp.SetCookie(w, &nethttp.Cookie{
		Name:     sessionCookie,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: nethttp.SameSiteLaxMode,
	})
}
