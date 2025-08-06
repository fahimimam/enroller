package utils

import (
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
)

type ErrorCode string

func IsDup(err error) bool {
	var psqlErr *pgconn.PgError
	return errors.As(err, &psqlErr) && psqlErr.Code == "23505"
}
