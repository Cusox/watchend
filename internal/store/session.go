package store

import (
	"context"
	"time"

	"github.com/cusox/watchend/internal/util"
)

type Session struct {
	ID                              int64
	UserID                          int64
	TokenHash                       []byte
	ExpiresAt, CreatedAt, UpdatedAt time.Time
}

func (s *Store) CreateSession(ctx context.Context, userID int64, tokenHash []byte, expiresAt time.Time) (Session, error) {
	now := s.now().UTC()

	result, err := s.db.ExecContext(ctx, `INSERT INTO sessions(user_id,token_hash,expires_at,created_at,updated_at) VALUES(?,?,?,?,?)`, userID, tokenHash, util.Unix(expiresAt), util.Unix(now), util.Unix(now))
	if err != nil {
		return Session{}, err
	}

	id, err := result.LastInsertId()

	return Session{
		ID:        id,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt.UTC(),
		CreatedAt: now,
		UpdatedAt: now,
	}, err
}

func (s *Store) DeleteSession(ctx context.Context, hash []byte) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash=?`, hash)

	return err
}

func (s *Store) SessionByTokenHash(ctx context.Context, hash []byte) (Session, error) {
	var v Session
	var expires, created, updated int64

	err := s.db.QueryRowContext(ctx, `SELECT id,user_id,token_hash,expires_at,created_at,updated_at FROM sessions WHERE token_hash=? AND expires_at>?`, hash, util.Unix(s.now())).Scan(&v.ID, &v.UserID, &v.TokenHash, &expires, &created, &updated)
	v.ExpiresAt = util.Timestamp(expires)
	v.CreatedAt = util.Timestamp(created)
	v.UpdatedAt = util.Timestamp(updated)

	return v, notFound(err)
}
