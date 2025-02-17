CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	balance INT DEFAULT 1000 CHECK(balance >= 0),
	username VARCHAR(255) UNIQUE NOT NULL,
	password VARCHAR(255) NOT NULL
);
CREATE TABLE items (
	id SERIAL PRIMARY KEY, 
	name VARCHAR(255) UNIQUE NOT NULL,
	price INT NOT NULL CHECK(price >=0)
);
CREATE TABLE user_items (
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    item_id INT REFERENCES items(id) ON DELETE CASCADE,
    quantity INT default 1 CHECK (quantity > 0),
    PRIMARY KEY (user_id, item_id)
);
CREATE INDEX idx_user_items_user_id ON user_items(user_id);
CREATE INDEX idx_user_items_item_id ON user_items(item_id);

CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    from_user_id INT REFERENCES users(id) ON DELETE CASCADE,
    to_user_id INT REFERENCES users(id) ON DELETE CASCADE,
    amount INT NOT NULL CHECK (amount > 0)
);
CREATE INDEX idx_transactions_from_user_id ON transactions(from_user_id);
CREATE INDEX idx_transactions_to_user_id ON transactions(to_user_id);


INSERT INTO items (name, price) VALUES ('t-shirt', 80);
INSERT INTO items (name, price) VALUES ('cup', 20);
INSERT INTO items (name, price) VALUES ('book', 50);
INSERT INTO items (name, price) VALUES ('pen', 10);
INSERT INTO items (name, price) VALUES ('powerbank', 200);
INSERT INTO items (name, price) VALUES ('hoody', 300);
INSERT INTO items (name, price) VALUES ('umbrella', 200);
INSERT INTO items (name, price) VALUES ('socks', 10);
INSERT INTO items (name, price) VALUES ('wallet', 50);
INSERT INTO items (name, price) VALUES ('pink-hoody', 500);
