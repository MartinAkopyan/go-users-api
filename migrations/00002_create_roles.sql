-- +goose Up
CREATE TABLE roles (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

INSERT INTO roles (name) VALUES ('user'), ('admin');

-- +goose Down
DROP TABLE roles;