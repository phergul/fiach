-- +goose Up
CREATE TABLE injection_targets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    target_relative_path TEXT NOT NULL COLLATE NOCASE,
    executable_relative_path TEXT NOT NULL,
    api_family TEXT NOT NULL CHECK (api_family IN ('directx', 'vulkan')),
    directx_api TEXT CHECK (directx_api IS NULL OR directx_api IN ('d3d9', 'd3d10', 'd3d11', 'd3d12')),
    architecture TEXT NOT NULL CHECK (architecture IN ('x86', 'x64')),
    primary_owner TEXT NOT NULL CHECK (primary_owner IN ('reshade', 'optiscaler')),
    primary_proxy_filename TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('managed', 'drifted', 'recovery_required')),
    recovery_journal_id TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_verified_at TEXT,
    UNIQUE(game_id, target_relative_path)
);

CREATE INDEX idx_injection_targets_game_id ON injection_targets(game_id);

CREATE TABLE injection_optiscaler (
    injection_target_id INTEGER PRIMARY KEY REFERENCES injection_targets(id) ON DELETE CASCADE,
    proxy_filename TEXT NOT NULL,
    dxgi_spoofing INTEGER NOT NULL CHECK (dxgi_spoofing IN (0, 1)),
    process_filter TEXT,
    release_tag TEXT NOT NULL,
    release_version TEXT NOT NULL,
    release_asset_name TEXT NOT NULL,
    release_digest TEXT NOT NULL,
    management_origin TEXT NOT NULL CHECK (management_origin IN ('installed', 'adopted')),
    manifest_json TEXT NOT NULL,
    warning_version TEXT NOT NULL,
    warning_acknowledged_at TEXT
);

CREATE TABLE injection_reshade (
    injection_target_id INTEGER PRIMARY KEY REFERENCES injection_targets(id) ON DELETE CASCADE,
    preferred_proxy_filename TEXT NOT NULL,
    active_runtime_filename TEXT NOT NULL,
    build_variant TEXT NOT NULL CHECK (build_variant IN ('standard', 'addon')),
    runtime_version TEXT NOT NULL,
    installer_tag TEXT,
    installer_asset_name TEXT,
    installer_url TEXT,
    installer_digest TEXT,
    installer_size INTEGER,
    management_origin TEXT NOT NULL CHECK (management_origin IN ('installed', 'adopted')),
    manifest_json TEXT NOT NULL
);

-- +goose StatementBegin
CREATE TRIGGER trg_injection_reshade_backfill_directx_api_insert
AFTER INSERT ON injection_reshade
WHEN (
    SELECT api_family = 'directx' AND directx_api IS NULL
    FROM injection_targets
    WHERE id = NEW.injection_target_id
)
BEGIN
    UPDATE injection_targets
    SET directx_api = CASE LOWER(NEW.preferred_proxy_filename)
        WHEN 'd3d9.dll' THEN 'd3d9'
        WHEN 'd3d10.dll' THEN 'd3d10'
        WHEN 'd3d10core.dll' THEN 'd3d10'
        WHEN 'd3d11.dll' THEN 'd3d11'
        WHEN 'd3d12.dll' THEN 'd3d12'
        WHEN 'dxgi.dll' THEN 'd3d11'
        ELSE directx_api
    END
    WHERE id = NEW.injection_target_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_injection_reshade_backfill_directx_api_update
AFTER UPDATE OF preferred_proxy_filename ON injection_reshade
WHEN (
    SELECT api_family = 'directx' AND directx_api IS NULL
    FROM injection_targets
    WHERE id = NEW.injection_target_id
)
BEGIN
    UPDATE injection_targets
    SET directx_api = CASE LOWER(NEW.preferred_proxy_filename)
        WHEN 'd3d9.dll' THEN 'd3d9'
        WHEN 'd3d10.dll' THEN 'd3d10'
        WHEN 'd3d10core.dll' THEN 'd3d10'
        WHEN 'd3d11.dll' THEN 'd3d11'
        WHEN 'd3d12.dll' THEN 'd3d12'
        WHEN 'dxgi.dll' THEN 'd3d11'
        ELSE directx_api
    END
    WHERE id = NEW.injection_target_id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS trg_injection_reshade_backfill_directx_api_update;
DROP TRIGGER IF EXISTS trg_injection_reshade_backfill_directx_api_insert;
DROP TABLE IF EXISTS injection_reshade;
DROP TABLE IF EXISTS injection_optiscaler;
DROP INDEX IF EXISTS idx_injection_targets_game_id;
DROP TABLE IF EXISTS injection_targets;
