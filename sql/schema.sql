DROP TABLE IF EXISTS posts, blocks, users;

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
