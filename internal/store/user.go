package store

import (
	"context"
	"time"

	"github.com/cusox/watchend/internal/util"
)

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

func (s *Store) CreateUser(ctx context.Context, username, passwordHash string) (User, error) {
	now := s.now()

	result, err := s.db.ExecContext(ctx, `INSERT INTO users(username,password_hash,created_at) VALUES(?,?,?)`, username, passwordHash, util.Unix(now))
	if err != nil {
		return User{}, err
	}

	id, err := result.LastInsertId()

	return User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    now,
	}, err
}

func (s *Store) UserByUsername(ctx context.Context, username string) (User, error) {
	var u User
	var created int64

	err := s.db.QueryRowContext(ctx, `SELECT id, username, password_hash, created_at FROM users WHERE username=?`, username).Scan(
		&u.ID,
		&u.Username,
		&u.PasswordHash,
		&created,
	)
	u.CreatedAt = util.Timestamp(created)

	return u, notFound(err)
}

func (s *Store) UpdatePasswordHash(ctx context.Context, userID int64, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET password_hash=? WHERE id=?`, passwordHash, userID)
	return err
}
