-- +goose Up
CREATE TABLE reshade_targets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    target_relative_path TEXT NOT NULL COLLATE NOCASE,
    executable_relative_path TEXT NOT NULL,
    rendering_api TEXT NOT NULL CHECK (rendering_api IN ('d3d9', 'd3d10', 'd3d11', 'd3d12')),
    proxy_filename TEXT NOT NULL,
    architecture TEXT NOT NULL CHECK (architecture IN ('x86', 'x64')),
    build_variant TEXT NOT NULL CHECK (build_variant IN ('standard', 'addon')),
    runtime_version TEXT NOT NULL,
    installer_tag TEXT,
    installer_asset_name TEXT,
    installer_url TEXT,
    installer_digest TEXT,
    installer_size INTEGER,
    management_origin TEXT NOT NULL CHECK (management_origin IN ('installed', 'adopted')),
    status TEXT NOT NULL CHECK (status IN ('managed', 'drifted', 'recovery_required')),
    manifest_json TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_verified_at TEXT,
    UNIQUE(game_id, target_relative_path)
);

CREATE INDEX idx_reshade_targets_game_id ON reshade_targets(game_id);

-- +goose Down
DROP INDEX IF EXISTS idx_reshade_targets_game_id;
DROP TABLE IF EXISTS reshade_targets;
