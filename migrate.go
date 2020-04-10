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
CREATE TABLE users (
		id SERIAL PRIMARY KEY,
		name VARCHAR UNIQUE NOT NULL,
		email VARCHAR UNIQUE NOT NULL
);

CREATE TABLE blocks (
		id SERIAL PRIMARY KEY,
		title VARCHAR
);

CREATE TABLE posts (
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
ALTER TABLE users ADD COLUMN created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE posts ADD COLUMN created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE blocks ADD COLUMN created_at TIMESTAMP DEFAULT NOW();
		`,
	},
	{
		name: "tokens table for auth",
		up: `
CREATE TABLE tokens (
	id SERIAL PRIMARY KEY,
	token VARCHAR UNIQUE NOT NULL
);
		`,
	},
	{
		name: "convert to timestamptz",
		up: `
		ALTER TABLE blocks ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'AEST';
		ALTER TABLE posts ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'AEST';
		ALTER TABLE users ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'AEST';
		ALTER TABLE migrations ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE 'AEST';
		`,
	},
}

func migrate(db *sql.DB) error {
	checkQuery := `SELECT EXISTS ( SELECT * FROM information_schema.tables WHERE information_schema.tables.table_name = 'migrations' );`
	row := db.QueryRow(checkQuery)

	exists := false
	if err := row.Scan(&exists); err != nil {
		return fmt.Errorf("couldn't check if `migrations` table existed: %w", err)
	}

	if !exists {
		log.Println("Creating migrations table")

		initQuery := `CREATE TABLE migrations ( id SERIAL PRIMARY KEY, name VARCHAR NOT NULL, created_at TIMESTAMP DEFAULT NOW() );`
		_, err := db.Exec(initQuery)

		if err != nil {
			return fmt.Errorf("couldn't create `migrations` table: %w", err)
		}
	}

	log.Println("Applying migrations")
	alreadyDoneQuery := `SELECT EXISTS (SELECT * FROM migrations WHERE name = $1 LIMIT 1 );`
	for _, m := range migrations {
		exists := false
		row := db.QueryRow(alreadyDoneQuery, m.name)
		if err := row.Scan(&exists); err != nil {
			return err
		}

		if exists {
			log.Println("Skipping migration:", m.name)

			continue
		}

		_, err := db.Exec(m.up)
		if err != nil {
			return fmt.Errorf("failed in migration %s: %w", m.name, err)
		}

		log.Println("Applied migration:", m.name)

		saveQuery := `INSERT INTO migrations (name) VALUES ($1);`
		_, err = db.Exec(saveQuery, m.name)
		if err != nil {
			return fmt.Errorf("couldn't save migration %s: %w", m.name, err)
		}
	}

	log.Println("Finished applying migrations")

	return nil
}
