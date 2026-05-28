-- +goose Up
CREATE TABLE games (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    install_path TEXT NOT NULL UNIQUE,
    source TEXT NOT NULL DEFAULT 'manual',
    source_id TEXT,
    available INTEGER NOT NULL DEFAULT 1 CHECK (available IN (0, 1)),
    last_seen_at TEXT,
    mod_storage_path TEXT,
    mod_storage_path_override TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE mods (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT 'folder' CHECK (source_type IN ('folder', 'archive')),
    source_path TEXT NOT NULL,
    original_source_path TEXT NOT NULL,
    original_source_name TEXT,
    file_count INTEGER,
    directory_count INTEGER,
    total_size_bytes INTEGER,
    metadata_json TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE mod_install_configs (
    mod_id INTEGER PRIMARY KEY REFERENCES mods(id) ON DELETE CASCADE,
    strategy_type TEXT NOT NULL CHECK (
        strategy_type IN (
            'generic_copy',
            'replace_files',
            'bepinex',
            'unreal_pak'
        )
    ),
    target_base TEXT NOT NULL DEFAULT 'game_root' CHECK (target_base IN ('game_root')),
    target_relative_path TEXT NOT NULL,
    source_subpath TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
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

CREATE TABLE applied_profile_states (
    game_id INTEGER PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE,
    profile_id INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    manifest_json TEXT NOT NULL,
    profile_snapshot_json TEXT NOT NULL,
    profile_snapshot_hash TEXT NOT NULL,
    profile_composition_snapshot_json TEXT,
    profile_composition_snapshot_hash TEXT,
    applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_mods_game_id ON mods(game_id);
CREATE UNIQUE INDEX idx_mods_game_original_source_path ON mods(game_id, original_source_path);
CREATE UNIQUE INDEX idx_games_source_source_id
ON games(source, source_id)
WHERE source_id IS NOT NULL;
CREATE INDEX idx_profiles_game_id ON profiles(game_id);
CREATE INDEX idx_profile_mods_mod_id ON profile_mods(mod_id);
CREATE INDEX idx_applied_profile_states_profile_id ON applied_profile_states(profile_id);

-- +goose Down
DROP INDEX IF EXISTS idx_applied_profile_states_profile_id;
DROP INDEX IF EXISTS idx_profile_mods_mod_id;
DROP INDEX IF EXISTS idx_profiles_game_id;
DROP INDEX IF EXISTS idx_games_source_source_id;
DROP INDEX IF EXISTS idx_mods_game_original_source_path;
DROP INDEX IF EXISTS idx_mods_game_id;

DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS applied_profile_states;
DROP TABLE IF EXISTS profile_mods;
DROP TABLE IF EXISTS profiles;
DROP TABLE IF EXISTS mod_install_configs;
DROP TABLE IF EXISTS mods;
DROP TABLE IF EXISTS games;
