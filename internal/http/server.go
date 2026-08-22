package http

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	nethttp "net/http"
	"time"

	"github.com/cusox/watchend/internal/github"
	"github.com/cusox/watchend/internal/store"
	templatefiles "github.com/cusox/watchend/web/templates"
)

const (
	maxBodyBytes = 1 << 20
)

type Store interface {
	UserByUsername(context.Context, string) (store.User, error)

	CreateSession(context.Context, int64, []byte, time.Time) (store.Session, error)
	DeleteSession(context.Context, []byte) error
	SessionByTokenHash(context.Context, []byte) (store.Session, error)

	ListRepositories(context.Context) ([]store.Repository, error)
	RepositoryByID(context.Context, int64) (store.Repository, error)
	RandomRepository(context.Context, string) (store.Repository, error)
	ListRepositoriesPage(context.Context, int, int, string, string) ([]store.Repository, bool, error)
	ListRepositoriesPageSearch(context.Context, int, int, string, string, string) ([]store.Repository, bool, error)
	UpdateRepositoryDetails(context.Context, int64, string, []string, []string) error
}

type Searcher interface {
	Search(context.Context, string, int) ([]store.Repository, error)
}

type Readiness interface {
	Ready(context.Context) error
}

type Options struct {
	Store         Store
	SessionSecret string
	SessionTTL    time.Duration
	SecureCookies bool
	Logger        *slog.Logger
	Searcher      Searcher
	Readiness     Readiness
	Syncer        Syncer
}

type Syncer interface {
	Sync(context.Context) error
}

type SyncStatus = github.SyncStatus

type ProgressSyncer interface {
	Start() error
}

type SyncStatusProvider interface {
	Status() SyncStatus
}

type Server struct {
	store     Store
	secret    []byte
	ttl       time.Duration
	secure    bool
	logger    *slog.Logger
	searcher  Searcher
	readiness Readiness
	templates *template.Template
	syncer    Syncer
}

func New(opts Options) (nethttp.Handler, error) {
	if opts.Store == nil {
		return nil, errors.New("http: store is required")
	}

	if len(opts.SessionSecret) < 32 {
		return nil, errors.New("http: session secret must be at least 32 bytes")
	}

	if opts.SessionTTL <= 0 {
		return nil, errors.New("http: session TTL must be positive")
	}

	t, err := template.New("").Funcs(template.FuncMap{"stars": formatStars}).ParseFS(templatefiles.FS, "*.html", "pages/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	s := &Server{
		store:     opts.Store,
		secret:    []byte(opts.SessionSecret),
		ttl:       opts.SessionTTL,
		secure:    opts.SecureCookies,
		logger:    opts.Logger,
		searcher:  opts.Searcher,
		readiness: opts.Readiness,
		templates: t,
		syncer:    opts.Syncer,
	}

	mux := nethttp.NewServeMux()
	mux.Handle("GET /static/", s.static())
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.ready)
	mux.HandleFunc("GET /login", s.loginPage)
	mux.HandleFunc("POST /login", s.login)
	mux.Handle("POST /logout", s.auth(nethttp.HandlerFunc(s.logout)))
	mux.Handle("GET /", s.auth(nethttp.HandlerFunc(s.repositories)))
	mux.Handle("GET /repositories", s.auth(nethttp.HandlerFunc(s.repositories)))
	mux.Handle("GET /repositories/more", s.auth(nethttp.HandlerFunc(s.moreRepositories)))
	mux.Handle("GET /repositories/random", s.auth(nethttp.HandlerFunc(s.randomRepository)))
	mux.Handle("GET /repositories/cards", s.auth(nethttp.HandlerFunc(s.repositoryCards)))
	mux.Handle("GET /repositories/{id}", s.auth(nethttp.HandlerFunc(s.repositoryCard)))
	mux.Handle("GET /repositories/{id}/edit", s.auth(nethttp.HandlerFunc(s.editRepository)))
	mux.Handle("POST /repositories/{id}", s.auth(nethttp.HandlerFunc(s.updateRepository)))
	mux.Handle("POST /repositories/sync", s.auth(nethttp.HandlerFunc(s.syncRepositories)))
	mux.Handle("GET /repositories/sync/status", s.auth(nethttp.HandlerFunc(s.syncStatus)))

	return s.logging(s.recovery(s.security(s.limitBody(s.csrf(mux))))), nil
}
