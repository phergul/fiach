package services

import (
	"strings"

	"github.com/phergul/fiach/internal/apperror"
)

func platformUserError(err error) error {
	if err == nil {
		return nil
	}
	if apperror.UserMessage(err) != "" {
		return err
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "ReShade management is only supported on Windows"),
		strings.Contains(message, "ReShade discovery is only supported on Windows"):
		return apperror.Wrap("ReShade management is only supported on Windows.", err)
	case strings.Contains(message, "OptiScaler management is only supported on Windows"):
		return apperror.Wrap("OptiScaler management is only supported on Windows.", err)
	case strings.Contains(message, "game install path is required"):
		return apperror.Wrap("This game does not have an install path configured.", err)
	case strings.Contains(message, "is not a directory"):
		return apperror.Wrap("The game install path is not a valid folder.", err)
	default:
		return err
	}
}
