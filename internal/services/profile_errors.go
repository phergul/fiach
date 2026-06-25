package services

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/storage"
)

func profileUserError(err error) error {
	if err == nil {
		return nil
	}
	if apperror.UserMessage(err) != "" {
		return err
	}

	switch {
	case errors.Is(err, storage.ErrDuplicateProfileName):
		return apperror.Wrap("A profile with this name already exists for this game.", err)
	case errors.Is(err, storage.ErrProfileNameRequired):
		return apperror.Wrap("Profile name is required.", err)
	case errors.Is(err, sql.ErrNoRows):
		return apperror.Wrap("Profile was not found.", err)
	default:
		return err
	}
}

func profilePlanUserError(err error) error {
	if err == nil {
		return nil
	}
	if apperror.UserMessage(err) != "" {
		return err
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "operation plan has blocking issues"):
		return apperror.Wrap("Fix the issues in the plan before applying.", err)
	case strings.Contains(message, "profile ID must be positive"):
		return apperror.Wrap("A valid profile must be selected.", err)
	case strings.Contains(message, "game ID must be positive"):
		return apperror.Wrap("A valid game must be selected.", err)
	case strings.Contains(message, "was not found") && strings.Contains(message, "profile"):
		return apperror.Wrap("Profile was not found.", err)
	case strings.Contains(message, "restore vanilla before applying another profile"):
		return apperror.Wrap("Restore vanilla before applying another profile.", err)
	case strings.Contains(message, "no applied profile state found"):
		return apperror.Wrap("No profile is currently applied for this game.", err)
	default:
		return profileUserError(err)
	}
}
