-- +goose Up
CREATE TABLE applied_file_states (
    game_id              INTEGER NOT NULL,
    game_relative_path   TEXT NOT NULL COLLATE NOCASE,
    profile_id           INTEGER NOT NULL,
    baseline_exists      INTEGER NOT NULL DEFAULT 0,
    baseline_sha256      TEXT,
    baseline_size_bytes  INTEGER,
    baseline_backup_path TEXT,
    applied_exists       INTEGER NOT NULL DEFAULT 0,
    applied_sha256       TEXT,
    applied_size_bytes   INTEGER,
    winning_source_kind  TEXT,
    winning_source_id    TEXT,
    winning_mod_id       INTEGER,
    winning_load_order   INTEGER,
    output_kind          TEXT NOT NULL DEFAULT 'copied',
    user_decision        TEXT,
    last_applied_at      TEXT NOT NULL,
    PRIMARY KEY (game_id, game_relative_path),
    FOREIGN KEY (game_id) REFERENCES applied_profile_states(game_id) ON DELETE CASCADE,
    FOREIGN KEY (profile_id) REFERENCES profiles(id)
);

CREATE INDEX idx_applied_file_states_profile_id ON applied_file_states(profile_id);

-- +goose Down
DROP INDEX IF EXISTS idx_applied_file_states_profile_id;
DROP TABLE IF EXISTS applied_file_states;
