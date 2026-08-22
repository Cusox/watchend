package main

import (
	"context"
	"crypto/subtle"
	"errors"

	"github.com/cusox/watchend/internal/auth"
	"github.com/cusox/watchend/internal/store"
	"github.com/cusox/watchend/internal/util"
)

func ensureAdmin(ctx context.Context, db *store.Store, username, password string) error {
	u, err := db.UserByUsername(ctx, username)
	if err == nil {
		if auth.VerifyHashPassword(u.PasswordHash, password) == nil {
			return nil
		}
		legacy := util.HashString(password)
		if subtle.ConstantTimeCompare([]byte(u.PasswordHash), legacy) != 1 {
			return errors.New("admin user exists with an invalid password")
		}
		hash, err := auth.GenerateHashPassword(password)
		if err != nil {
			return err
		}
		return db.UpdatePasswordHash(ctx, u.ID, hash)
	}

	if !errors.Is(err, store.ErrNotFound) {
		return err
	}

	hash, err := auth.GenerateHashPassword(password)
	if err != nil {
		return err
	}

	_, err = db.CreateUser(ctx, username, hash)

	return err
}
