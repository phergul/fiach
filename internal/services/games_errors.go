package services

import (
	"strings"

	"github.com/phergul/fiach/internal/apperror"
)

func gamesUserError(err error) error {
	if err == nil {
		return nil
	}
	if apperror.UserMessage(err) != "" {
		return err
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "game sources are not configured"):
		return apperror.Wrap("Game sources are not configured.", err)
	case strings.Contains(message, "game source is not configured"):
		return apperror.Wrap("A game source is not configured.", err)
	case strings.Contains(message, "game source identifier is required"):
		return apperror.Wrap("A game source identifier is required.", err)
	default:
		return err
	}
}
