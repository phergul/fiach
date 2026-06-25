package services

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/storage"
)

func modUserError(err error) error {
	if err == nil {
		return nil
	}
	if apperror.UserMessage(err) != "" {
		return err
	}

	switch {
	case errors.Is(err, storage.ErrDuplicateTagName):
		return apperror.Wrap("A tag with this name already exists for this game.", err)
	case errors.Is(err, sql.ErrNoRows):
		return apperror.Wrap("Mod was not found.", err)
	case strings.Contains(err.Error(), "mod name is required"):
		return apperror.Wrap("Mod name is required.", err)
	case strings.Contains(err.Error(), "managed mod storage path is required"):
		return apperror.Wrap("Managed mod storage is not configured for this game.", err)
	case strings.Contains(err.Error(), "managed mod source path is required"):
		return apperror.Wrap("This mod does not have a managed source path.", err)
	case strings.Contains(err.Error(), "is outside managed storage"):
		return apperror.Wrap("This mod is stored outside managed mod storage and cannot be deleted safely.", err)
	default:
		return err
	}
}

func modImportUserError(err error) error {
	if err == nil {
		return nil
	}
	if apperror.UserMessage(err) != "" {
		return err
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "import source type is required"):
		return apperror.Wrap("Choose an import source type.", err)
	case strings.Contains(message, "unsupported import source type"):
		return apperror.Wrap("That import source type is not supported.", err)
	case strings.Contains(message, "game ID must be positive"):
		return apperror.Wrap("A valid game must be selected.", err)
	case strings.Contains(message, "mod name is required"):
		return apperror.Wrap("Mod name is required.", err)
	case strings.Contains(message, "mod ID must be positive"):
		return apperror.Wrap("A valid mod must be selected.", err)
	case strings.Contains(message, "update source is required"), strings.Contains(message, "import source is required"):
		return apperror.Wrap("Choose a source to import.", err)
	case strings.Contains(message, "source folder") && strings.Contains(message, "is empty"):
		return apperror.Wrap("The selected source folder is empty.", err)
	case strings.Contains(message, "archive is empty"):
		return apperror.Wrap("The selected archive is empty.", err)
	case strings.Contains(message, "password-protected archives are not supported"):
		return apperror.Wrap("Password-protected archives are not supported.", err)
	case strings.Contains(message, "multipart archives are not supported"):
		return apperror.Wrap("Multipart archives are not supported.", err)
	case strings.Contains(message, "archive format was not recognized"):
		return apperror.Wrap("The archive format was not recognized.", err)
	default:
		return err
	}
}
