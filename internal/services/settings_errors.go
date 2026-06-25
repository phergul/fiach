package services

import (
	"strings"

	"github.com/phergul/fiach/internal/apperror"
)

func settingsUserError(err error) error {
	if err == nil {
		return nil
	}
	if apperror.UserMessage(err) != "" {
		return err
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "theme ID is required"):
		return apperror.Wrap("Theme is required.", err)
	default:
		return err
	}
}
