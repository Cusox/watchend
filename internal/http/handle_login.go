package http

import (
	nethttp "net/http"
	"strings"
	"time"

	"github.com/cusox/watchend/internal/auth"
	"github.com/cusox/watchend/internal/util"
)

func (s *Server) loginPage(w nethttp.ResponseWriter, r *nethttp.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if _, err = s.store.SessionByTokenHash(r.Context(), util.HashString(c.Value)); err == nil {
			nethttp.Redirect(w, r, "/", nethttp.StatusSeeOther)
			return
		}
	}

	s.render(w, "login", pageData{
		Title:     "Log in",
		CSRFToken: csrfToken(r),
	})
}

func (s *Server) loginError(w nethttp.ResponseWriter, r *nethttp.Request) {
	w.WriteHeader(nethttp.StatusUnauthorized)

	s.render(w, "login", pageData{
		Title:     "Log in",
		CSRFToken: csrfToken(r),
		Error:     "Invalid username or password.",
	})
}

func (s *Server) login(w nethttp.ResponseWriter, r *nethttp.Request) {
	username := strings.TrimSpace(r.PostForm.Get("username"))
	password := r.PostForm.Get("password")
	if username == "" || len(username) > 128 || password == "" || len(password) > 1024 {
		s.loginError(w, r)
		return
	}

	u, err := s.store.UserByUsername(r.Context(), username)
	if err != nil || auth.VerifyHashPassword(u.PasswordHash, password) != nil {
		s.loginError(w, r)
		return
	}

	token, hash, err := auth.NewSessionToken()
	if err != nil {
		nethttp.Error(w, "internal server error", 500)
		return
	}

	if _, err = s.store.CreateSession(r.Context(), u.ID, hash, time.Now().Add(s.ttl)); err != nil {
		nethttp.Error(w, "internal server error", 500)
		return
	}

	s.setSession(w, token)

	nethttp.Redirect(w, r, "/", nethttp.StatusSeeOther)
}

func (s *Server) logout(w nethttp.ResponseWriter, r *nethttp.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		_ = s.store.DeleteSession(r.Context(), util.HashString(c.Value))
	}

	s.clearSession(w)

	nethttp.Redirect(w, r, "/login", nethttp.StatusSeeOther)
}
