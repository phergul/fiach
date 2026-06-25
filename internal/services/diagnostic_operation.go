package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/phergul/fiach/internal/diagnostics"
)

type diagnosticOperation struct {
	ctx       context.Context
	logger    *slog.Logger
	operation string
	startedAt time.Time
	attrs     []slog.Attr
}

func startDiagnosticOperation(ctx context.Context, logger *slog.Logger, operation string, message string, attrs ...slog.Attr) diagnosticOperation {
	if logger == nil {
		logger = slog.Default()
	}

	startedAt := time.Now()
	baseAttrs := append([]slog.Attr{
		slog.String("operation", operation),
		slog.String("event", diagnostics.EventStarted),
	}, attrs...)

	logger.LogAttrs(ctx, slog.LevelInfo, message, baseAttrs...)

	return diagnosticOperation{
		ctx:       ctx,
		logger:    logger,
		operation: operation,
		startedAt: startedAt,
		attrs:     append([]slog.Attr{}, attrs...),
	}
}

func (op diagnosticOperation) complete(message string, attrs ...slog.Attr) {
	op.logger.LogAttrs(op.ctx, slog.LevelInfo, message, op.eventAttrs(diagnostics.EventCompleted, attrs...)...)
}

func (op diagnosticOperation) infoEvent(event string, message string, attrs ...slog.Attr) {
	op.logger.LogAttrs(op.ctx, slog.LevelInfo, message, op.eventAttrs(event, attrs...)...)
}

func (op diagnosticOperation) warn(message string, attrs ...slog.Attr) {
	op.logger.LogAttrs(op.ctx, slog.LevelWarn, message, op.eventAttrs(diagnostics.EventCompleted, attrs...)...)
}

func (op diagnosticOperation) warnEvent(event string, message string, attrs ...slog.Attr) {
	op.logger.LogAttrs(op.ctx, slog.LevelWarn, message, op.eventAttrs(event, attrs...)...)
}

func logOperationEvent(ctx context.Context, logger *slog.Logger, level slog.Level, operation string, event string, message string, attrs ...slog.Attr) {
	if logger == nil {
		logger = slog.Default()
	}

	baseAttrs := append([]slog.Attr{
		slog.String("operation", operation),
		slog.String("event", event),
	}, attrs...)
	logger.LogAttrs(ctx, level, message, baseAttrs...)
}

func (op diagnosticOperation) fail(message string, err error, attrs ...slog.Attr) {
	op.logger.LogAttrs(op.ctx, slog.LevelError, message, op.eventAttrs(diagnostics.EventFailed, append(attrs, diagnostics.ErrorAttr(err))...)...)
}

func (op diagnosticOperation) failWithMappedError(message string, err error, mapUserError func(error) error) error {
	if err == nil {
		return nil
	}
	if mapUserError != nil {
		err = mapUserError(err)
	}
	op.fail(message, err)
	return err
}

func (op diagnosticOperation) eventAttrs(event string, attrs ...slog.Attr) []slog.Attr {
	result := append([]slog.Attr{
		slog.String("operation", op.operation),
		slog.String("event", event),
	}, op.attrs...)

	result = append(result, attrs...)
	result = append(result, diagnostics.DurationAttr(op.startedAt))

	return result
}
