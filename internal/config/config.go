package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAddr          = ":3000"
	defaultDatabasePath  = "/data/watchend.db"
	defaultAdminUsername = "admin"
	defaultAdminPassword = ""
	defaultSyncInterval  = 6 * time.Hour
	defaultSessionTTL    = 30 * 24 * time.Hour
)

type Config struct {
	Addr          string
	DatabasePath  string
	AdminUsername string
	AdminPassword string
	SyncInterval  time.Duration
	SessionTTL    time.Duration
	SecureCookies bool
	GitHubToken   string
}

func Load() (Config, error) {
	lookup := os.LookupEnv

	cfg := Config{
		Addr:          defaultAddr,
		DatabasePath:  defaultDatabasePath,
		AdminUsername: defaultAdminUsername,
		AdminPassword: defaultAdminPassword,
		SyncInterval:  defaultSyncInterval,
		SessionTTL:    defaultSessionTTL,
		SecureCookies: true,
		GitHubToken:   "",
	}

	if value, ok := lookup("WATCHEND_ADDR"); ok && strings.TrimSpace(value) != "" {
		cfg.Addr = strings.TrimSpace(value)
	}

	if value, ok := lookup("WATCHEND_DATABASE_PATH"); ok && strings.TrimSpace(value) != "" {
		cfg.DatabasePath = strings.TrimSpace(value)
	}

	if value, ok := lookup("WATCHEND_ADMIN_USERNAME"); ok && strings.TrimSpace(value) != "" {
		cfg.AdminUsername = strings.TrimSpace(value)
	}

	if value, ok := lookup("WATCHEND_ADMIN_PASSWORD"); ok && strings.TrimSpace(value) != "" {
		cfg.AdminPassword = strings.TrimSpace(value)
	}
	if cfg.AdminPassword == "" {
		return Config{}, errors.New("WATCHEND_ADMIN_PASSWORD must not be empty")
	}

	if raw, ok := lookup("WATCHEND_SYNC_INTERVAL"); ok && strings.TrimSpace(raw) != "" {
		interval, err := time.ParseDuration(raw)
		if err != nil || interval <= 0 {
			return Config{}, errors.New("WATCHEND_SYNC_INTERVAL must be a positive duration")
		}
		cfg.SyncInterval = interval
	}

	if value, ok := lookup("WATCHEND_SECURE_COOKIES"); ok {
		secureCookies, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, errors.New("WATCHEND_SECURE_COOKIES must be a boolean")
		}
		cfg.SecureCookies = secureCookies
	}

	cfg.GitHubToken, _ = lookup("WATCHEND_GITHUB_TOKEN")
	if cfg.GitHubToken == "" {
		return Config{}, fmt.Errorf("missing required environment variables: %s", "WATCHEND_GITHUB_TOKEN")
	}

	return cfg, nil
}
