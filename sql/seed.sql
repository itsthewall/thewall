INSERT INTO users (name, email) VALUES ('admin', 'admin@obanana.rocks');

INSERT INTO blocks (title) VALUES ('Test Block');

INSERT INTO posts (block_id, user_id, title, body) VALUES (1, 1, 'Test 1', 'Body');
INSERT INTO posts (block_id, user_id, title, body) VALUES (1, 1, 'Test 2', 'Body');
