package http

import (
	"fmt"
	"html/template"
	nethttp "net/http"
	"strconv"
	"strings"

	"github.com/cusox/watchend/internal/store"
)

func repositoryDirection(value string) string {
	if value == "asc" {
		return "asc"
	}
	return "desc"
}

func repositorySort(value string) string {
	switch value {
	case "stars", "updated", "name", "starred":
		return value
	default:
		return "stars"
	}
}

const repositoryPageSize = 24

func (s *Server) randomRepository(w nethttp.ResponseWriter, r *nethttp.Request) {
	repo, err := s.store.RandomRepository(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		if err == store.ErrNotFound {
			nethttp.Error(w, "no matching repositories", nethttp.StatusNotFound)
			return
		}
		nethttp.Error(w, "internal server error", nethttp.StatusInternalServerError)
		return
	}
	nethttp.Redirect(w, r, repo.HTMLURL, nethttp.StatusFound)
}

func (s *Server) repositories(w nethttp.ResponseWriter, r *nethttp.Request) {
	sort := repositorySort(r.URL.Query().Get("sort"))
	direction := repositoryDirection(r.URL.Query().Get("direction"))
	search := r.URL.Query().Get("q")
	repos, hasMore, err := s.store.ListRepositoriesPageSearch(r.Context(), repositoryPageSize, 0, sort, direction, search)
	if err != nil {
		nethttp.Error(w, "internal server error", 500)
		return
	}

	s.render(w, "repositories", pageData{
		Title: "Repositories", CSRFToken: csrfToken(r), Repositories: repos, Query: sort, Direction: direction, Search: r.URL.Query().Get("q"),
		RepositoryOffset: len(repos), RepositoryHasMore: hasMore,
	})
}

func (s *Server) repositoryCards(w nethttp.ResponseWriter, r *nethttp.Request) {
	sort := repositorySort(r.URL.Query().Get("sort"))
	direction := repositoryDirection(r.URL.Query().Get("direction"))
	search := r.URL.Query().Get("q")
	repos, hasMore, err := s.store.ListRepositoriesPageSearch(r.Context(), repositoryPageSize, 0, sort, direction, search)
	if err != nil {
		nethttp.Error(w, "internal server error", 500)
		return
	}
	s.render(w, "repository-cards", pageData{Repositories: repos, Query: sort, Direction: direction, Search: search, RepositoryOffset: len(repos), RepositoryHasMore: hasMore})
}

func (s *Server) moreRepositories(w nethttp.ResponseWriter, r *nethttp.Request) {
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil || offset < 0 || offset > 1000000 {
		nethttp.Error(w, "invalid offset", 400)
		return
	}
	sort := repositorySort(r.URL.Query().Get("sort"))
	direction := repositoryDirection(r.URL.Query().Get("direction"))
	search := r.URL.Query().Get("q")
	repos, hasMore, err := s.store.ListRepositoriesPageSearch(r.Context(), repositoryPageSize, offset, sort, direction, search)
	if err != nil {
		nethttp.Error(w, "internal server error", 500)
		return
	}
	s.render(w, "repository-cards", pageData{Repositories: repos, Query: sort, Direction: direction, Search: search, RepositoryOffset: offset + len(repos), RepositoryHasMore: hasMore})
}

func splitLabels(value string) []string {
	var result []string
	seen := map[string]bool{}
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" && len(item) <= 64 && !seen[item] {
			result = append(result, item)
			seen[item] = true
		}
	}
	return result
}

func (s *Server) repositoryCard(w nethttp.ResponseWriter, r *nethttp.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		nethttp.Error(w, "invalid repository ID", 400)
		return
	}
	repo, err := s.store.RepositoryByID(r.Context(), id)
	if err != nil {
		nethttp.NotFound(w, r)
		return
	}
	s.render(w, "repository-card", pageData{Repositories: []store.Repository{repo}})
}

func (s *Server) editRepository(w nethttp.ResponseWriter, r *nethttp.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		nethttp.Error(w, "invalid repository ID", 400)
		return
	}
	repo, err := s.store.RepositoryByID(r.Context(), id)
	if err != nil {
		nethttp.NotFound(w, r)
		return
	}
	s.render(w, "repository-edit", pageData{Repositories: []store.Repository{repo}, CSRFToken: csrfToken(r)})
}

func (s *Server) updateRepository(w nethttp.ResponseWriter, r *nethttp.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	note := strings.TrimSpace(r.PostForm.Get("note"))
	if err != nil || id <= 0 || len(note) > 4000 {
		nethttp.Error(w, "invalid repository details", 400)
		return
	}
	if err = s.store.UpdateRepositoryDetails(r.Context(), id, note, splitLabels(r.PostForm.Get("categories")), splitLabels(r.PostForm.Get("tags"))); err != nil {
		nethttp.Error(w, "internal server error", 500)
		return
	}
	repo, err := s.store.RepositoryByID(r.Context(), id)
	if err != nil {
		nethttp.NotFound(w, r)
		return
	}
	s.render(w, "repository-card", pageData{Repositories: []store.Repository{repo}})
}

func (s *Server) syncRepositories(w nethttp.ResponseWriter, r *nethttp.Request) {
	progress, ok := s.syncer.(ProgressSyncer)
	if !ok {
		nethttp.Error(w, "sync unavailable", nethttp.StatusNotImplemented)
		return
	}
	if err := progress.Start(); err != nil {
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<p class=\"error\">Sync is already running.</p>"))
			return
		}
		nethttp.Error(w, err.Error(), nethttp.StatusConflict)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<span data-sync-running="true">Sync started.</span>`))
		return
	}
	nethttp.Redirect(w, r, "/", nethttp.StatusSeeOther)
}

func (s *Server) writeSyncStatus(w nethttp.ResponseWriter, status SyncStatus) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status.Running {
		if status.Total > 0 {
			_, _ = fmt.Fprintf(w, `<span data-sync-running="true" hx-get="/repositories/sync/status" hx-trigger="every 1s" hx-target="this" hx-swap="outerHTML" hx-indicator="#sync-progress">Syncing repositories: %d / %d</span>`, status.Current, status.Total)
		} else {
			_, _ = w.Write([]byte(`<span data-sync-running="true" hx-get="/repositories/sync/status" hx-trigger="every 1s" hx-target="this" hx-swap="outerHTML" hx-indicator="#sync-progress">Fetching starred repositories…</span>`))
		}
		return
	}
	if status.Error != "" {
		_, _ = fmt.Fprintf(w, `<span class="error">Sync failed: %s</span>`, template.HTMLEscapeString(status.Error))
		return
	}
	w.Header().Set("HX-Trigger", "repositories-updated")
	_, _ = w.Write([]byte(`<span class="notice" data-sync-done="true">Repositories synchronized.</span>`))
}

func (s *Server) syncStatus(w nethttp.ResponseWriter, r *nethttp.Request) {
	provider, ok := s.syncer.(SyncStatusProvider)
	if !ok {
		nethttp.Error(w, "sync status unavailable", 501)
		return
	}
	s.writeSyncStatus(w, provider.Status())
}
