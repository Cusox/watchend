package http

import (
	nethttp "net/http"
	"strconv"
	"strings"

	"github.com/cusox/watchend/internal/store"
)

type pageData struct {
	Title, CSRFToken, Error, Message, Query, Direction, Search string
	// Dashboard                               store.Dashboard
	Repositories, SearchResults []store.Repository
	RepositoryOffset            int
	RepositoryHasMore           bool
}

func formatStars(value int) string {
	if value < 1000 {
		return strconv.Itoa(value)
	}
	formatted := strconv.FormatFloat(float64(value)/1000, 'f', 1, 64)
	formatted = strings.TrimSuffix(formatted, ".0")
	return formatted + "K"
}

func (s *Server) render(w nethttp.ResponseWriter, name string, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		s.logger.Error("render template", "template", name, "error", err)
	}
}
