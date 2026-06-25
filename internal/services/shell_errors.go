package services

import (
	"strings"

	"github.com/phergul/fiach/internal/apperror"
)

func shellUserError(err error) error {
	if err == nil {
		return nil
	}
	if apperror.UserMessage(err) != "" {
		return err
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "application is not configured"):
		return apperror.Wrap("The application is not configured.", err)
	case strings.Contains(message, "export path is required"):
		return apperror.Wrap("Choose a path to export logs to.", err)
	case strings.Contains(message, "directory path is required"):
		return apperror.Wrap("Directory path is required.", err)
	case strings.Contains(message, "does not exist"):
		return apperror.Wrap("That directory does not exist.", err)
	case strings.Contains(message, "is not a directory"):
		return apperror.Wrap("That path is not a directory.", err)
	default:
		return err
	}
}
