package main

import (
	"database/sql"
	"fmt"
	"log"
)

// migration describes a SQL operation which gets ran everytime the server is started. This is _not_ the right way to do this forever, but it works for now.
//
// here are some examples on writing idempotent migrations in postgres: https://gist.github.com/michelmilezzi/8f30607cdf9389ea35ff7548bb0226fe
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
DO $$
BEGIN
	ALTER TABLE users ADD COLUMN created_at TIMESTAMP;
EXCEPTION WHEN duplicate_column THEN
	RAISE NOTICE 'Field already exists. Ignoring...';
END$$;
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
