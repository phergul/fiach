package applyplan

import (
	"github.com/phergul/fiach/internal/fileops"
)

type Context struct {
	GameInstallPath    string
	GameModStoragePath string
}

type operationOutcome struct {
	message          string
	createdDirectory bool
}

var computeFileIntegrity = fileops.FileIntegrity
