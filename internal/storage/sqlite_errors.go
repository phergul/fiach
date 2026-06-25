package storage

import (
	"errors"
	"fmt"
	"strings"
)

type uniqueConstraint struct {
	table   string
	columns []string
	target  error
}

var uniqueConstraintMatchers = []uniqueConstraint{
	{
		table:   "profiles",
		columns: []string{"game_id", "name"},
		target:  ErrDuplicateProfileName,
	},
	{
		table:   "tags",
		columns: []string{"game_id", "normalized_name"},
		target:  ErrDuplicateTagName,
	},
}

func mapSQLiteError(err error) error {
	if err == nil {
		return nil
	}

	for _, matcher := range uniqueConstraintMatchers {
		if isUniqueConstraint(err, matcher.table, matcher.columns...) {
			return fmt.Errorf("%w: %w", matcher.target, err)
		}
	}

	return err
}

func isUniqueConstraint(err error, table string, columns ...string) bool {
	if err == nil || table == "" || len(columns) == 0 {
		return false
	}

	message := strings.ToLower(sqliteErrorMessage(err))
	if !strings.Contains(message, "unique constraint failed") {
		return false
	}

	table = strings.ToLower(strings.TrimSpace(table))
	if !strings.Contains(message, table) {
		return false
	}

	for _, column := range columns {
		if !strings.Contains(message, strings.ToLower(strings.TrimSpace(column))) {
			return false
		}
	}

	return true
}

func sqliteErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	message := strings.TrimSpace(err.Error())
	for unwrapped := errors.Unwrap(err); unwrapped != nil; unwrapped = errors.Unwrap(unwrapped) {
		candidate := strings.TrimSpace(unwrapped.Error())
		if candidate != "" {
			message = candidate
		}
	}

	return message
}
