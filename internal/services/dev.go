package services

import (
	"context"

	"github.com/phergul/fiach/internal/appmode"
	"github.com/phergul/fiach/internal/devlog"
	"github.com/phergul/fiach/internal/services/dto"
)

const defaultDevLogLimit = 500

type DevService struct {
	databasePath string
}

func NewDevService(databasePath string) *DevService {
	return &DevService{
		databasePath: databasePath,
	}
}

func (s *DevService) IsDevMode(context.Context) bool {
	return appmode.IsDev()
}

func (s *DevService) GetDevInfo(context.Context) dto.DevInfo {
	return dto.DevInfo{
		DataDir:      appmode.DataRoot(),
		DatabasePath: s.databasePath,
	}
}

func (s *DevService) ListDevLogs(_ context.Context, limit int) []dto.DevLogEntry {
	if limit <= 0 {
		limit = defaultDevLogLimit
	}

	logs := devlog.List(limit)
	result := make([]dto.DevLogEntry, len(logs))
	for index, entry := range logs {
		result[index] = dto.DevLogEntry{
			Timestamp: entry.Timestamp,
			Message:   entry.Message,
		}
	}

	return result
}

func (s *DevService) ClearDevLogs(context.Context) {
	devlog.Clear()
}
