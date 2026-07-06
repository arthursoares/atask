package service

import (
	"database/sql"
	"errors"
	"time"

	"github.com/atask/atask/internal/domain"
)

func timeNow() time.Time {
	return time.Now().UTC()
}

// mapNotFound converts a sql.ErrNoRows (or any error wrapping it) into
// domain.ErrNotFound. Used by cross-entity ownership checks (spec §2.4) to
// signal that a foreign-key reference does not belong to the requesting
// user (or does not exist at all) without leaking which case it was.
func mapNotFound(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}
