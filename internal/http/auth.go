package http

import (
	nethttp "net/http"

	"github.com/cusox/watchend/internal/util"
)

func (s *Server) auth(next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil {
			s.unauth(w, r)
			return
		}

		_, err = s.store.SessionByTokenHash(r.Context(), util.HashString(c.Value))
		if err != nil {
			s.clearSession(w)
			s.unauth(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) unauth(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(nethttp.StatusUnauthorized)
		return
	}

	nethttp.Redirect(w, r, "/login", nethttp.StatusSeeOther)
}
