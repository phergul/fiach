-- +goose Up
ALTER TABLE games ADD COLUMN source TEXT NOT NULL DEFAULT 'manual';
ALTER TABLE games ADD COLUMN source_id TEXT;
ALTER TABLE games ADD COLUMN available INTEGER NOT NULL DEFAULT 1 CHECK (available IN (0, 1));
ALTER TABLE games ADD COLUMN last_seen_at TEXT;

CREATE UNIQUE INDEX idx_games_source_source_id
ON games(source, source_id)
WHERE source_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_games_source_source_id;

CREATE TABLE games_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    install_path TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO games_old (id, name, install_path, created_at, updated_at)
SELECT id, name, install_path, created_at, updated_at
FROM games;

DROP TABLE games;
ALTER TABLE games_old RENAME TO games;
