package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/cusox/watchend/internal/util"
)

type Repository struct {
	ID                                                                                    int64
	Owner, Name, FullName, Description, ReadMe, HTMLURL, DefaultBranch, Language, License string
	Stars                                                                                 int
	Archived                                                                              bool
	Topics, Categories, Tags                                                              []string
	Note                                                                                  string
	CreatedAt, UpdatedAt, StarredAt                                                       time.Time
}

func (s *Store) UpsertRepository(ctx context.Context, r Repository) error {
	topics := strings.Join(r.Topics, ",")
	categories := strings.Join(r.Categories, ",")
	tags := strings.Join(r.Tags, ",")
	_, err := s.db.ExecContext(ctx, `INSERT INTO repositories(id,owner,name,full_name,description,readme,html_url,default_branch,stars,archived,topics,note,created_at,updated_at,starred_at,language,license,categories,tags) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET owner=excluded.owner,name=excluded.name,full_name=excluded.full_name,description=excluded.description,html_url=excluded.html_url,default_branch=excluded.default_branch,stars=excluded.stars,archived=excluded.archived,topics=excluded.topics,updated_at=excluded.updated_at,starred_at=excluded.starred_at,language=excluded.language,license=excluded.license`, r.ID, r.Owner, r.Name, r.FullName, r.Description, r.ReadMe, r.HTMLURL, r.DefaultBranch, r.Stars, util.BoolInt(r.Archived), topics, r.Note, util.Unix(r.CreatedAt), util.Unix(r.UpdatedAt), util.Unix(r.StarredAt), r.Language, r.License, categories, tags)
	return err
}

func (s *Store) DeleteRepositoriesExcept(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		_, err := s.db.ExecContext(ctx, `DELETE FROM repositories`)
		return err
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM repositories WHERE id NOT IN (`+placeholders+`)`, args...)
	return err
}

func (s *Store) UpdateRepositoryDetails(ctx context.Context, id int64, note string, categories, tags []string) error {
	if id <= 0 {
		return requirePositive("repository ID", id)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE repositories SET note=?,categories=?,tags=? WHERE id=?`, note, strings.Join(categories, ","), strings.Join(tags, ","), id)
	return err
}

func (s *Store) RandomRepository(ctx context.Context, search string) (Repository, error) {
	search = strings.TrimSpace(search)
	query := `SELECT id,owner,name,full_name,description,readme,html_url,default_branch,stars,archived,topics,note,created_at,updated_at,starred_at,language,license,categories,tags FROM repositories`
	args := []any{}
	if search != "" {
		query += ` WHERE lower(full_name || ' ' || description || ' ' || language || ' ' || license || ' ' || categories || ' ' || tags || ' ' || note) LIKE lower(?)`
		args = append(args, "%"+search+"%")
	}
	query += ` ORDER BY RANDOM() LIMIT 1`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return Repository{}, err
	}
	defer rows.Close()
	items, err := scanRepositories(rows)
	if err != nil {
		return Repository{}, err
	}
	if len(items) == 0 {
		return Repository{}, ErrNotFound
	}
	return items[0], nil
}

func (s *Store) RepositoryByID(ctx context.Context, id int64) (Repository, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,owner,name,full_name,description,readme,html_url,default_branch,stars,archived,topics,note,created_at,updated_at,starred_at,language,license,categories,tags FROM repositories WHERE id=?`, id)
	if err != nil {
		return Repository{}, err
	}
	defer rows.Close()
	items, err := scanRepositories(rows)
	if err != nil {
		return Repository{}, err
	}
	if len(items) == 0 {
		return Repository{}, ErrNotFound
	}
	return items[0], nil
}
func (s *Store) ListRepositoriesPage(ctx context.Context, limit, offset int, sort, direction string) ([]Repository, bool, error) {
	return s.ListRepositoriesPageSearch(ctx, limit, offset, sort, direction, "")
}

func (s *Store) ListRepositoriesPageSearch(ctx context.Context, limit, offset int, sort, direction, search string) ([]Repository, bool, error) {
	if limit <= 0 {
		limit = 24
	}
	if offset < 0 {
		offset = 0
	}
	column := map[string]string{"stars": "stars", "updated": "updated_at", "name": "full_name", "starred": "starred_at"}[sort]
	if column == "" {
		column = "stars"
	}
	if direction != "asc" {
		direction = "desc"
	}
	query := `SELECT id,owner,name,full_name,description,readme,html_url,default_branch,stars,archived,topics,note,created_at,updated_at,starred_at,language,license,categories,tags FROM repositories`
	args := []any{}
	if search = strings.TrimSpace(search); search != "" {
		query += ` WHERE lower(full_name || ' ' || description || ' ' || language || ' ' || license || ' ' || categories || ' ' || tags || ' ' || note) LIKE lower(?)`
		args = append(args, "%"+search+"%")
	}
	query += ` ORDER BY ` + column + " " + strings.ToUpper(direction) + `, full_name ASC, id ASC LIMIT ? OFFSET ?`
	args = append(args, limit+1, offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	all, err := scanRepositories(rows)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(all) > limit
	if hasMore {
		all = all[:limit]
	}
	return all, hasMore, nil
}

func (s *Store) ListRepositories(ctx context.Context) ([]Repository, error) {
	repos, _, err := s.ListRepositoriesPage(ctx, 0, 0, "stars", "desc")
	return repos, err
}

func scanRepositories(rows *sql.Rows) ([]Repository, error) {
	var repos []Repository
	for rows.Next() {
		var r Repository
		var archived int
		var topics, categories, tags string
		var created, updated, starred int64
		if err := rows.Scan(&r.ID, &r.Owner, &r.Name, &r.FullName, &r.Description, &r.ReadMe, &r.HTMLURL, &r.DefaultBranch, &r.Stars, &archived, &topics, &r.Note, &created, &updated, &starred, &r.Language, &r.License, &categories, &tags); err != nil {
			return nil, err
		}
		r.Archived = archived != 0
		if topics != "" {
			r.Topics = strings.Split(topics, ",")
		}
		if categories != "" {
			r.Categories = strings.Split(categories, ",")
		}
		if tags != "" {
			r.Tags = strings.Split(tags, ",")
		}
		r.CreatedAt, r.UpdatedAt, r.StarredAt = util.Timestamp(created), util.Timestamp(updated), util.Timestamp(starred)
		repos = append(repos, r)
	}
	return repos, rows.Err()
}
