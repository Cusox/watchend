package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cusox/watchend/internal/store"
)

var ErrAlreadyRunning = errors.New("sync already running")

type SyncStatus struct {
	Running bool
	Current int
	Total   int
	Done    bool
	Error   string
}

type Syncer struct {
	db     *store.Store
	client *Client
	mu     sync.Mutex
	status SyncStatus
}

func New(token string, db *store.Store) *Syncer {
	return &Syncer{db: db, client: NewClient(&http.Client{Timeout: 30 * time.Second}, token)}
}

func (s *Syncer) Start() error {
	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		return ErrAlreadyRunning
	}
	s.status = SyncStatus{Running: true}
	s.mu.Unlock()
	go s.run(context.Background())
	return nil
}

func (s *Syncer) Status() SyncStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *Syncer) Sync(ctx context.Context) error {
	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		return ErrAlreadyRunning
	}
	s.status = SyncStatus{Running: true}
	s.mu.Unlock()
	return s.run(ctx)
}

func (s *Syncer) run(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, 10*time.Minute)
	defer cancel()
	var processed int
	var repositoryIDs []int64
	err := s.client.StarsEach(ctx, func(stars []Star, total int) error {
		s.mu.Lock()
		s.status.Total = total
		s.mu.Unlock()
		for _, star := range stars {
			r := star.Repository
			repositoryIDs = append(repositoryIDs, r.ID)
			if err := s.db.UpsertRepository(ctx, store.Repository{ID: r.ID, Owner: r.Owner.Login, Name: r.Name, FullName: r.FullName, Description: r.Description, HTMLURL: r.HTMLURL, DefaultBranch: r.DefaultBranch, Stars: r.Stars, Archived: r.Archived, Topics: r.Topics, Language: r.Language, License: r.License.SPDXID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, StarredAt: star.StarredAt}); err != nil {
				return fmt.Errorf("save repository %s: %w", r.FullName, err)
			}
			processed++
			s.mu.Lock()
			s.status.Current = processed
			if s.status.Total < processed {
				s.status.Total = processed
			}
			s.mu.Unlock()
		}
		return nil
	})
	if err == nil {
		err = s.db.DeleteRepositoriesExcept(ctx, repositoryIDs)
	}
	s.finish(err)
	return err
}

func (s *Syncer) finish(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.Running = false
	s.status.Done = err == nil
	if err != nil {
		s.status.Error = err.Error()
	}
}
