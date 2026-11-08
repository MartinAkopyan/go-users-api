package postgres

import (
	"context"

	"github.com/MartinAkopyan/go-users-api/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (repo *UserRepo) Create(ctx context.Context, u domain.User) (domain.User, error) {

	err := repo.pool.QueryRow(ctx,
		"INSERT INTO users (full_name, email, password_hash, role_id) "+
			"SELECT $1, $2, $3, id FROM roles WHERE name = $4 "+
			"RETURNING id, created_at, updated_at", u.FullName, u.Email, u.PasswordHash, string(u.Role),
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		return domain.User{}, mapDBError(err)
	}

	return u, nil
}

func (repo *UserRepo) GetByID(ctx context.Context, id int64) (domain.User, error) {
	var u domain.User

	err := repo.pool.QueryRow(ctx, `
		select u.id, u.full_name, u.email, u.password_hash,
		r.name, u.created_at, u.updated_at
		from users u
		join roles r on u.role_id = r.id
		where u.id = $1`, id,
	).Scan(&u.ID, &u.FullName, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		return domain.User{}, mapDBError(err)
	}

	return u, nil
}
