-- +goose Up
CREATE TABLE
    items (
        market_hash_name TEXT PRIMARY KEY,
        classid TEXT NOT NULL,
        type TEXT,
        commodity BOOLEAN NOT NULL DEFAULT false,
        tradable BOOLEAN NOT NULL DEFAULT true,
        icon_url TEXT,
        name_color TEXT,
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW ()
    );

-- +goose Down
DROP TABLE items;