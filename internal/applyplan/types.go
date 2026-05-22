package applyplan

import (
	"github.com/phergul/mod-manager/internal/fileops"
)

type Context struct {
	GameInstallPath    string
	GameModStoragePath string
}

type resolvedContext struct {
	gameInstallPath    string
	gameModStoragePath string
}

type operationOutcome struct {
	message          string
	createdDirectory bool
}

var computeFileIntegrity = fileops.FileIntegrity
