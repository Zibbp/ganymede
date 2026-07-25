package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	osExec "os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/zibbp/ganymede/internal/worker"
)

const (
	crashWorkerHelperEnv      = "GANYMEDE_TEST_CRASH_WORKER_HELPER"
	crashWorkerReadyPathEnv   = "GANYMEDE_TEST_CRASH_WORKER_READY_PATH"
	crashWorkerStartupTimeout = 45 * time.Second
)

// CrashableWorker is a real worker process owned by an integration test. A
// process boundary is required because cancelling or stopping an in-process
// River client is graceful and therefore cannot reproduce an OS-level crash.
type CrashableWorker struct {
	cmd      *osExec.Cmd
	waitDone chan struct{}
	waitErr  error
}

// StartCrashableWorker launches the current package's test binary and asks its
// named helper test to run only the worker. The package using this helper must
// define:
//
//	func TestWorkerCrashHelper(t *testing.T) { tests.RunWorkerCrashHelper(t) }
func StartCrashableWorker(t *testing.T, helperTestName string) *CrashableWorker {
	t.Helper()

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve test executable: %v", err)
	}

	readyPath := filepath.Join(t.TempDir(), "worker-ready")
	cmd := osExec.Command(executable, "-test.run=^"+helperTestName+"$", "-test.count=1")
	cmd.Env = append(os.Environ(),
		crashWorkerHelperEnv+"=1",
		crashWorkerReadyPathEnv+"="+readyPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start crashable worker: %v", err)
	}

	process := &CrashableWorker{
		cmd:      cmd,
		waitDone: make(chan struct{}),
	}
	go func() {
		process.waitErr = cmd.Wait()
		close(process.waitDone)
	}()

	t.Cleanup(func() {
		_ = process.Crash()
	})

	deadline := time.Now().Add(crashWorkerStartupTimeout)
	for {
		if _, err := os.Stat(readyPath); err == nil {
			return process
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stat crashable worker readiness file: %v", err)
		}

		select {
		case <-process.waitDone:
			t.Fatalf("crashable worker exited before becoming ready: %v", process.result())
		default:
		}

		if time.Now().After(deadline) {
			_ = process.Crash()
			t.Fatalf("timed out waiting for crashable worker to start")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// Crash sends SIGKILL through os.Process.Kill and waits for the process to
// exit. This deliberately bypasses River and Ganymede's graceful shutdown.
func (p *CrashableWorker) Crash() error {
	select {
	case <-p.waitDone:
		return nil
	default:
	}

	killErr := p.cmd.Process.Kill()
	<-p.waitDone
	if killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
		return fmt.Errorf("kill crashable worker: %w", killErr)
	}
	return nil
}

func (p *CrashableWorker) result() error {
	return p.waitErr
}

// RunWorkerCrashHelper turns the current test binary into a standalone worker
// process. In normal test execution it returns immediately; only
// StartCrashableWorker sets the sentinel environment variable.
func RunWorkerCrashHelper(t *testing.T) {
	t.Helper()
	if os.Getenv(crashWorkerHelperEnv) != "1" {
		return
	}

	ctx := context.Background()
	workerClient, err := worker.SetupWorker(ctx)
	if err != nil {
		t.Fatalf("set up crashable worker: %v", err)
	}
	defer func() {
		_ = workerClient.Close()
	}()

	if err := workerClient.Start(); err != nil {
		t.Fatalf("start crashable worker: %v", err)
	}

	readyPath := os.Getenv(crashWorkerReadyPathEnv)
	if readyPath == "" {
		t.Fatal("crashable worker readiness file is not configured")
	}
	if err := os.WriteFile(readyPath, []byte("ready"), 0o600); err != nil {
		t.Fatalf("write crashable worker readiness file: %v", err)
	}

	select {}
}
