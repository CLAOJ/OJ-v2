package executors

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

// runSandboxed executes a command in a sandboxed environment.
// This is a basic implementation - for production, use seccomp-bpf.
func runSandboxed(ctx context.Context, command string, input string, limits RunLimits) (*RunResult, error) {
	var cmd *exec.Cmd

	// Check if command contains spaces (e.g., "java Main")
	// For a more robust solution, parse the command properly
	cmd = exec.CommandContext(ctx, "sh", "-c", command)

	// Set up resource limits
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Set process group
		Setpgid: true,
	}

	// Set up I/O pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return &RunResult{
			Status: StatusInternalError,
			Error:  err.Error(),
		}, nil
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return &RunResult{
			Status: StatusInternalError,
			Error:  err.Error(),
		}, nil
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return &RunResult{
			Status: StatusInternalError,
			Error:  err.Error(),
		}, nil
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return &RunResult{
			Status: StatusRuntimeError,
			Error:  err.Error(),
		}, nil
	}

	// Write input
	go func() {
		stdin.Write([]byte(input))
		stdin.Close()
	}()

	// Read output with limit
	outputCh := make(chan []byte, 1)
	errCh := make(chan error, 1)

	go func() {
		buf := make([]byte, limits.OutputSize)
		n, err := stdout.Read(buf)
		if err != nil && err.Error() != "EOF" {
			errCh <- err
			return
		}
		outputCh <- buf[:n]
	}()

	// Read stderr
	errOutput, _ := readAllLimited(stderr, 64*1024)

	// Wait for completion or timeout
	doneCh := make(chan error, 1)
	go func() {
		doneCh <- cmd.Wait()
	}()

	select {
	case output := <-outputCh:
		select {
		case err := <-doneCh:
			// Normal exit
			exitCode := 0
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}

			result := &RunResult{
				ExitCode: exitCode,
				Output:   output,
				Error:    string(errOutput),
				Status:   StatusAccepted,
			}

			if exitCode != 0 {
				result.Status = StatusRuntimeError
			}

			return result, nil

		case <-time.After(limits.Time):
			// Time limit exceeded
			killProcess(cmd)
			return &RunResult{
				Status: StatusTimeLimitExceeded,
			}, nil
		}

	case err := <-errCh:
		return &RunResult{
			Status: StatusRuntimeError,
			Error:  err.Error(),
		}, nil

	case <-time.After(limits.Time):
		killProcess(cmd)
		return &RunResult{
			Status: StatusTimeLimitExceeded,
		}, nil
	}
}

// readAllLimited reads from a reader with a size limit.
func readAllLimited(r interface{ Read([]byte) (int, error) }, limit int64) ([]byte, error) {
	buf := make([]byte, limit)
	n, _ := r.Read(buf)
	return buf[:n], nil
}

// killProcess kills a process and its process group.
func killProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		// Kill entire process group
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
