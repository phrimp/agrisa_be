package utils

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type ExecType int

const (
	ExecInsert ExecType = iota
	ExecUpdate
	ExecDelete
)

func ExecWithCheck(db *sqlx.DB, query string, execType ExecType, args ...any) error {
	result, err := db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	// if Insert operation, don't need to check rows affected
	if execType == ExecInsert {
		return nil
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows affected")
	}

	return nil
}
