-- +goose Up
CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    normalized_name TEXT NOT NULL,
    color TEXT NOT NULL CHECK (
        color IN (
            'red',
            'orange',
            'yellow',
            'green',
            'teal',
            'blue',
            'purple',
            'pink'
        )
    ),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(game_id, normalized_name)
);

CREATE TABLE mod_tags (
    mod_id INTEGER NOT NULL REFERENCES mods(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(mod_id, tag_id)
);

CREATE INDEX idx_tags_game_id ON tags(game_id);
CREATE INDEX idx_mod_tags_tag_id ON mod_tags(tag_id);

-- +goose Down
DROP INDEX IF EXISTS idx_mod_tags_tag_id;
DROP INDEX IF EXISTS idx_tags_game_id;
DROP TABLE IF EXISTS mod_tags;
DROP TABLE IF EXISTS tags;
