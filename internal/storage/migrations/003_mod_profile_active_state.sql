-- +goose NO TRANSACTION

-- +goose Up
ALTER TABLE profiles ADD COLUMN is_active INTEGER NOT NULL DEFAULT 0 CHECK (is_active IN (0, 1));

CREATE UNIQUE INDEX idx_profiles_active_game_id
ON profiles(game_id)
WHERE is_active = 1;

-- +goose Down
PRAGMA foreign_keys = OFF;

DROP INDEX IF EXISTS idx_profiles_active_game_id;

CREATE TABLE profiles_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(game_id, name)
);

INSERT INTO profiles_old (id, game_id, name, created_at, updated_at)
SELECT id, game_id, name, created_at, updated_at
FROM profiles;

DROP TABLE profiles;
ALTER TABLE profiles_old RENAME TO profiles;

CREATE INDEX idx_profiles_game_id ON profiles(game_id);

PRAGMA foreign_key_check;
PRAGMA foreign_keys = ON;
