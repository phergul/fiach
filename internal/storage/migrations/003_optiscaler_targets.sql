-- +goose Up
CREATE TABLE optiscaler_targets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    target_relative_path TEXT NOT NULL COLLATE NOCASE,
    executable_relative_path TEXT NOT NULL,
    graphics_api TEXT NOT NULL CHECK (graphics_api IN ('directx', 'vulkan')),
    proxy_filename TEXT NOT NULL,
    dxgi_spoofing INTEGER NOT NULL CHECK (dxgi_spoofing IN (0, 1)),
    process_filter TEXT,
    release_tag TEXT NOT NULL,
    release_version TEXT NOT NULL,
    release_asset_name TEXT NOT NULL,
    release_digest TEXT NOT NULL,
    management_origin TEXT NOT NULL CHECK (management_origin IN ('installed', 'adopted')),
    status TEXT NOT NULL CHECK (status IN ('managed', 'drifted', 'recovery_required')),
    manifest_json TEXT NOT NULL,
    warning_version TEXT NOT NULL,
    warning_acknowledged_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_verified_at TEXT,
    UNIQUE(game_id, target_relative_path)
);

CREATE INDEX idx_optiscaler_targets_game_id ON optiscaler_targets(game_id);

-- +goose Down
DROP INDEX IF EXISTS idx_optiscaler_targets_game_id;
DROP TABLE IF EXISTS optiscaler_targets;
