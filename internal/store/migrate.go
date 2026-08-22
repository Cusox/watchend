package store

import (
	"context"
	"fmt"
)

const schemaVersion = 3

var versionSchema = `CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER PRIMARY KEY, 
	applied_at INTEGER NOT NULL
) STRICT`

var userSchema = []string{
	`CREATE TABLE users (
		id INTEGER PRIMARY KEY, 
		username TEXT NOT NULL UNIQUE, 
		password_hash TEXT NOT NULL, 
		created_at INTEGER NOT NULL
	) STRICT`,
	`CREATE TABLE sessions (
		id INTEGER PRIMARY KEY, 
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, 
		token_hash BLOB NOT NULL UNIQUE CHECK(length(token_hash)=32), 
		expires_at INTEGER NOT NULL, 
		created_at INTEGER NOT NULL, 
		updated_at INTEGER NOT NULL
	) STRICT`,
	`CREATE INDEX sessions_expires_at_idx ON sessions(expires_at)`,
}

var repositorySchema = []string{
	`CREATE TABLE repositories (
		id INTEGER PRIMARY KEY, 
		owner TEXT NOT NULL, 
		name TEXT NOT NULL, 
		full_name TEXT NOT NULL UNIQUE,
		description TEXT NOT NULL DEFAULT '', 
		readme TEXT NOT NULL DEFAULT '',
		html_url TEXT NOT NULL DEFAULT '', 
		default_branch TEXT NOT NULL DEFAULT '', 
		stars INTEGER NOT NULL DEFAULT 0, 
		archived INTEGER NOT NULL DEFAULT 0 CHECK(archived IN (0,1)), 
		topics TEXT NOT NULL DEFAULT '',
		note TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL, 
		updated_at INTEGER NOT NULL,
		starred_at INTEGER NOT NULL,
		language TEXT NOT NULL DEFAULT '',
		license TEXT NOT NULL DEFAULT '',
		categories TEXT NOT NULL DEFAULT '',
		tags TEXT NOT NULL DEFAULT ''
	) STRICT`,
	`CREATE VIRTUAL TABLE repositories_fts USING fts5(
		full_name,
		description, 
		readme, 
		topics, 
		note, 
		content='repositories', 
		content_rowid='id',
		tokenize='trigram'
	)`,
	`CREATE TRIGGER repositories_fts_insert AFTER INSERT ON repositories 
	BEGIN 
		INSERT INTO repositories_fts(
			rowid,
			full_name,
			description,
			readme,
			topics,
			note
		) VALUES(
			new.id,
			new.full_name,
			new.description,
			new.readme,
			new.topics,
			new.note
		);
	END`,
	`CREATE TRIGGER repositories_fts_delete AFTER DELETE ON repositories 
	BEGIN 
		INSERT INTO repositories_fts(
			repositories_fts,
			rowid,
			full_name,
			description,
			readme,
			topics,
			note
		) VALUES(
			'delete',
			old.id,
			old.full_name,
			old.description,
			old.readme,
			old.topics,
			old.note
		); 
	END`,
	`CREATE TRIGGER repositories_fts_update AFTER UPDATE OF 
		full_name,
		description,
		readme,
		topics,
		note
		ON repositories 
	BEGIN 
		INSERT INTO repositories_fts(
			repositories_fts,
			rowid,
			full_name,
			description,
			readme,
			topics,
			note
		) VALUES(
			'delete',
			old.id,
			old.full_name,
			old.description,
			old.readme,
			old.topics,
			old.note
		); 
		INSERT INTO repositories_fts(
			rowid,
			full_name,
			description,
			readme,
			topics,
			note
		) VALUES(
			new.id,
			new.full_name,
			new.description,
			new.readme,
			new.topics,
			new.note
		); 
	END`,
}

func (s *Store) Migrate(ctx context.Context) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err = conn.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return err
	}
	var foreignKeys int
	if err = conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		return fmt.Errorf("enable foreign keys: %w", err)
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(ctx, versionSchema); err != nil {
		return err
	}

	var version int
	err = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(version),0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return err
	}

	if version > schemaVersion {
		return fmt.Errorf("database schema version %d is newer than supported version %d", version, schemaVersion)
	}

	if version == 0 {
		for _, statement := range append(userSchema, repositorySchema...) {
			if _, err = tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("migrate schema: %w", err)
			}
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations(version, applied_at) VALUES (3, unixepoch())"); err != nil {
			return err
		}
		version = 3
	}
	if version == 1 {
		for _, statement := range []string{`ALTER TABLE repositories ADD COLUMN language TEXT NOT NULL DEFAULT ''`, `ALTER TABLE repositories ADD COLUMN license TEXT NOT NULL DEFAULT ''`} {
			if _, err = tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("migrate schema: %w", err)
			}
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations(version, applied_at) VALUES (2, unixepoch())"); err != nil {
			return err
		}
		version = 2
	}
	if version == 2 {
		for _, statement := range []string{`ALTER TABLE repositories ADD COLUMN categories TEXT NOT NULL DEFAULT ''`, `ALTER TABLE repositories ADD COLUMN tags TEXT NOT NULL DEFAULT ''`} {
			if _, err = tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("migrate schema: %w", err)
			}
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations(version, applied_at) VALUES (3, unixepoch())"); err != nil {
			return err
		}
	}
	return tx.Commit()
}
