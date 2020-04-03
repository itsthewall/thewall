package main

import (
	"database/sql"
	"fmt"
	"log"
)

type migration struct {
	name string
	up   string
}

var migrations []migration = []migration{
	{
		name: "init schema",
		up: `
CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name VARCHAR UNIQUE NOT NULL,
		email VARCHAR UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS blocks (
		id SERIAL PRIMARY KEY,
		title VARCHAR
);

CREATE TABLE IF NOT EXISTS posts (
		id SERIAL PRIMARY key,

		block_id INTEGER REFERENCES blocks(id),
		user_id INTEGER REFERENCES users(id),

		title VARCHAR,
		body VARCHAR
);
		`,
	},
	{
		name: "add created_at timestamps",
		up: `
BEGIN;

ALTER TABLE users ADD created_at TIMESTAMP NOT NULL DEFAULT NOW();
ALTER TABLE blocks ADD created_at TIMESTAMP NOT NULL DEFAULT NOW();
ALTER TABLE posts ADD created_at TIMESTAMP NOT NULL DEFAULT NOW();

COMMIT;
		`,
	},
}

func migrate(db *sql.DB) error {
	log.Println("Applying migrations")
	for _, m := range migrations {
		_, err := db.Exec(m.up)

		if err != nil {
			return fmt.Errorf("failed in migration %s: %w", m.name, err)
		}

		log.Println("Applied migration", m.name)
	}

	log.Println("Finished applying migrations")

	return nil
}
