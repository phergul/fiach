//go:build windows

package reshade

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type platformSetupProcessRunner struct{}

type limitedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (buffer *limitedBuffer) Write(contents []byte) (int, error) {
	originalLength := len(contents)
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining <= 0 {
		buffer.truncated = buffer.truncated || originalLength > 0
		return originalLength, nil
	}
	if len(contents) > remaining {
		contents = contents[:remaining]
		buffer.truncated = true
	}
	_, err := buffer.buffer.Write(contents)
	return originalLength, err
}

func (buffer *limitedBuffer) String() string {
	return buffer.buffer.String()
}

func (platformSetupProcessRunner) RunSetupProcess(
	ctx context.Context,
	request setupProcessRequest,
) (result setupProcessResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("run managed ReShade setup process: %w", err)
		}
	}()
	timeout := request.Timeout
	if timeout <= 0 {
		timeout = DefaultSetupTimeout
	}
	outputLimit := request.OutputLimit
	if outputLimit <= 0 {
		outputLimit = DefaultSetupOutputLimitBytes
	}
	runContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	stdout := &limitedBuffer{limit: outputLimit}
	stderr := &limitedBuffer{limit: outputLimit}
	command := exec.Command(request.ExecutablePath, request.Arguments...)
	command.Dir = request.WorkingDir
	command.Stdout = stdout
	command.Stderr = stderr
	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	result.StartedAt = time.Now()
	if err := command.Start(); err != nil {
		result.FinishedAt = time.Now()
		result.ExitCode = -1
		result.ElevationNeeded = errors.Is(err, syscall.Errno(740))
		result.Stdout = stdout.String()
		result.Stderr = stderr.String()
		return result, err
	}

	job, err := createKillOnCloseJob()
	if err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		result.FinishedAt = time.Now()
		result.ExitCode = -1
		return result, err
	}
	defer windows.CloseHandle(job)
	processHandle, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_INFORMATION,
		false,
		uint32(command.Process.Pid),
	)
	if err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		result.FinishedAt = time.Now()
		result.ExitCode = -1
		return result, fmt.Errorf("open ReShade setup process: %w", err)
	}
	assignErr := windows.AssignProcessToJobObject(job, processHandle)
	_ = windows.CloseHandle(processHandle)
	if assignErr != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		result.FinishedAt = time.Now()
		result.ExitCode = -1
		return result, fmt.Errorf("assign ReShade setup process to job: %w", assignErr)
	}

	waitResult := make(chan error, 1)
	go func() {
		waitResult <- command.Wait()
	}()
	select {
	case waitErr := <-waitResult:
		result.FinishedAt = time.Now()
		result.ExitCode = command.ProcessState.ExitCode()
		result.Stdout = stdout.String()
		result.Stderr = stderr.String()
		result.StdoutTruncated = stdout.truncated
		result.StderrTruncated = stderr.truncated
		if waitErr != nil {
			var exitError *exec.ExitError
			if errors.As(waitErr, &exitError) {
				return result, nil
			}
			return result, waitErr
		}
		return result, nil
	case <-runContext.Done():
		_ = windows.TerminateJobObject(job, 1)
		<-waitResult
		result.FinishedAt = time.Now()
		result.ExitCode = command.ProcessState.ExitCode()
		result.Stdout = stdout.String()
		result.Stderr = stderr.String()
		result.StdoutTruncated = stdout.truncated
		result.StderrTruncated = stderr.truncated
		result.TimedOut = errors.Is(runContext.Err(), context.DeadlineExceeded) &&
			ctx.Err() == nil
		result.Cancelled = !result.TimedOut
		return result, runContext.Err()
	}
}

func createKillOnCloseJob() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, fmt.Errorf("create ReShade setup process job: %w", err)
	}
	information := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	information.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&information)),
		uint32(unsafe.Sizeof(information)),
	); err != nil {
		_ = windows.CloseHandle(job)
		return 0, fmt.Errorf("configure ReShade setup process job: %w", err)
	}
	return job, nil
}
