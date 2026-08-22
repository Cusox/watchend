package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cusox/watchend/internal/util"
)

const sessionSecretFile = "session-secret"

func LoadOrCreateSessionSecret(databasePath string) ([]byte, error) {
	if databasePath == "" {
		return nil, errors.New("database path must not be empty")
	}

	path := filepath.Join(filepath.Dir(databasePath), sessionSecretFile)

	secret, err := os.ReadFile(path)
	if err == nil {
		if len(secret) < 32 {
			return nil, fmt.Errorf("session secret file %s is too short", path)
		}
		return secret, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	secret = make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return nil, err
	}

	if err = os.WriteFile(path, secret, 0600); err != nil {
		return nil, fmt.Errorf("create session secret: %w", err)
	}

	return secret, nil
}

func NewSessionToken() (token string, hash []byte, err error) {
	raw := make([]byte, tokenBytes)

	if _, err = rand.Read(raw); err != nil {
		return "", nil, err
	}

	token = base64.RawURLEncoding.EncodeToString(raw)

	return token, util.HashString(token), nil
}
