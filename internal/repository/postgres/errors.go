package postgres

import (
	"errors"

	"github.com/MartinAkopyan/go-users-api/internal/domain"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func mapDBError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return domain.ErrAlreadyExists
		case pgerrcode.ForeignKeyViolation:
			return domain.ErrNotFound
		}
	}

	return err
}
