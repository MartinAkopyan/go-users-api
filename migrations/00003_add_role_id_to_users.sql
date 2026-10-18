-- +goose Up
ALTER TABLE users ADD role_id BIGINT NOT NULL REFERENCES roles(id);



-- +goose Down
ALTER TABLE users DROP COLUMN role_id;