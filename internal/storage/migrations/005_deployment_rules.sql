-- +goose Up
CREATE TABLE deployment_rules (
    id                 INTEGER PRIMARY KEY,
    profile_id         INTEGER NOT NULL,
    game_relative_path TEXT NOT NULL COLLATE NOCASE,
    rule_kind          TEXT NOT NULL,
    winner_mod_id      INTEGER,
    explanation        TEXT,
    created_at         TEXT NOT NULL,
    UNIQUE (profile_id, game_relative_path, rule_kind),
    FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE
);

CREATE INDEX idx_deployment_rules_profile_id ON deployment_rules(profile_id);

-- +goose Down
DROP INDEX IF EXISTS idx_deployment_rules_profile_id;
DROP TABLE IF EXISTS deployment_rules;
