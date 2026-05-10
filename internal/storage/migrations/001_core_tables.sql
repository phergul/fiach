-- +goose Up
CREATE TABLE games (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    install_path TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE mods (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    source_path TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(game_id, source_path)
);

CREATE TABLE profiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(game_id, name)
);

CREATE TABLE profile_mods (
    profile_id INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    mod_id INTEGER NOT NULL REFERENCES mods(id) ON DELETE CASCADE,
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    load_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(profile_id, mod_id)
);

CREATE TABLE applied_manifests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id INTEGER NOT NULL,
    mod_id INTEGER NOT NULL,
    source_path TEXT NOT NULL,
    destination_path TEXT NOT NULL,
    checksum TEXT,
    file_size INTEGER,
    applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(profile_id, mod_id) REFERENCES profile_mods(profile_id, mod_id) ON DELETE CASCADE,
    UNIQUE(profile_id, destination_path)
);

CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_mods_game_id ON mods(game_id);
CREATE INDEX idx_profiles_game_id ON profiles(game_id);
CREATE INDEX idx_profile_mods_mod_id ON profile_mods(mod_id);
CREATE INDEX idx_applied_manifests_profile_mod ON applied_manifests(profile_id, mod_id);

-- +goose Down
DROP INDEX IF EXISTS idx_applied_manifests_profile_mod;
DROP INDEX IF EXISTS idx_profile_mods_mod_id;
DROP INDEX IF EXISTS idx_profiles_game_id;
DROP INDEX IF EXISTS idx_mods_game_id;

DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS applied_manifests;
DROP TABLE IF EXISTS profile_mods;
DROP TABLE IF EXISTS profiles;
DROP TABLE IF EXISTS mods;
DROP TABLE IF EXISTS games;
