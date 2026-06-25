package apperror

import (
	"errors"
	"fmt"
	"strings"
)

type UserError struct {
	Message string
	Err     error
}

func (e *UserError) Error() string {
	return e.Message
}

func (e *UserError) Unwrap() error {
	return e.Err
}

func New(message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return errors.New("something went wrong")
	}

	return &UserError{Message: message}
}

func Wrap(message string, err error) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return err
	}
	if err == nil {
		return New(message)
	}

	return &UserError{
		Message: message,
		Err:     err,
	}
}

func UserMessage(err error) string {
	if err == nil {
		return ""
	}

	var userErr *UserError
	if errors.As(err, &userErr) && strings.TrimSpace(userErr.Message) != "" {
		return userErr.Message
	}

	return ""
}

func Detail(err error) string {
	if err == nil {
		return ""
	}

	segments := make([]string, 0, 4)
	seen := make(map[string]struct{})

	for current := err; current != nil; current = errors.Unwrap(current) {
		segment := errorSegment(current)
		if segment == "" {
			continue
		}

		if _, exists := seen[segment]; exists {
			continue
		}
		seen[segment] = struct{}{}
		segments = append(segments, segment)
	}

	return strings.Join(segments, ": ")
}

func errorSegment(err error) string {
	var userErr *UserError
	if errors.As(err, &userErr) {
		return strings.TrimSpace(userErr.Message)
	}

	unwrapped := errors.Unwrap(err)
	if unwrapped == nil {
		return strings.TrimSpace(err.Error())
	}

	full := strings.TrimSpace(err.Error())
	wrapped := strings.TrimSpace(unwrapped.Error())
	if wrapped != "" && strings.HasSuffix(full, wrapped) {
		prefix := strings.TrimSpace(strings.TrimSuffix(full, wrapped))
		prefix = strings.TrimSuffix(prefix, ":")
		prefix = strings.TrimSpace(prefix)
		if prefix != "" {
			return prefix
		}

		return ""
	}

	return full
}

func IsUserError(err error) bool {
	var userErr *UserError
	return errors.As(err, &userErr)
}

func FormatDetail(err error) string {
	if detail := Detail(err); detail != "" {
		return detail
	}

	return fmt.Sprintf("%v", err)
}
