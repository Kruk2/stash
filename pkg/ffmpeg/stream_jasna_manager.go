package ffmpeg

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	stashExec "github.com/stashapp/stash/pkg/exec"
	"github.com/stashapp/stash/pkg/logger"
)

const (
	jasnaBaseURL     = "http://localhost:8765"
	jasnaIdleTimeout = 3 * time.Minute
)

type jasnaStreamManager struct {
	parentCtx context.Context
	client    *http.Client

	mu     sync.Mutex
	cancel context.CancelFunc
	cmd    *exec.Cmd
	done   chan struct{}

	currentArgs  string
	currentFile  string
	lastActivity time.Time
}

func newJasnaStreamManager(ctx context.Context) *jasnaStreamManager {
	m := &jasnaStreamManager{
		parentCtx: ctx,
		client:    &http.Client{Timeout: 60 * time.Second},
	}
	go m.idleMonitor()
	return m
}

func (m *jasnaStreamManager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callStop()
	m.stopProcessLocked()
}

func (m *jasnaStreamManager) stopProcessLocked() {
	m.currentFile = ""
	m.currentArgs = ""
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if m.done != nil {
		select {
		case <-m.done:
		case <-time.After(5 * time.Second):
		}
		m.done = nil
	}
	m.cmd = nil
}

func (m *jasnaStreamManager) ensureRunningLocked(variantArgs []string) error {
	argsKey := strings.Join(variantArgs, "\x00")
	if m.cmd != nil {
		select {
		case <-m.done:
			m.stopProcessLocked()
		default:
			if m.currentArgs == argsKey {
				return nil
			}
			logger.Infof("[jasna] variant changed, restarting process")
			m.callStop()
			m.stopProcessLocked()
		}
	}

	m.stopProcessLocked()

	procCtx, cancel := context.WithCancel(m.parentCtx)
	exe, cmdArgs := resolveJasnaCliCommand()
	args := append(cmdArgs, "--stream", "--no-browser", "--log-level", "info")
	args = append(args, variantArgs...)
	cmd := stashExec.CommandContext(procCtx, exe, args...)

	if wd := os.Getenv("JASNA_WORKING_DIR"); wd != "" {
		cmd.Dir = wd
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("jasna stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("jasna stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("jasna start: %w", err)
	}

	done := make(chan struct{})
	m.cancel = cancel
	m.cmd = cmd
	m.done = done
	m.lastActivity = time.Now()

	go m.consumeOutput("stdout", stdout, done)
	go m.consumeOutput("stderr", stderr, done)
	go m.wait(cmd, done)

	if err := m.waitForHealthy(procCtx); err != nil {
		m.stopProcessLocked()
		return fmt.Errorf("jasna not healthy: %w", err)
	}

	m.currentArgs = argsKey
	return nil
}

func (m *jasnaStreamManager) waitForHealthy(ctx context.Context) error {
	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return fmt.Errorf("timeout waiting for jasna-cli to become healthy")
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, jasnaBaseURL+"/status", nil)
			if err != nil {
				continue
			}
			resp, err := m.client.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
	}
}

func (m *jasnaStreamManager) wait(cmd *exec.Cmd, done chan struct{}) {
	err := cmd.Wait()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) && !errors.Is(err, context.Canceled) {
		logger.Errorf("[jasna] process exited: %v", err)
	}
	close(done)
}

func (m *jasnaStreamManager) consumeOutput(label string, r io.Reader, done <-chan struct{}) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			logger.Infof("[jasna:%s] %s", label, line)
		}
		select {
		case <-done:
			return
		default:
		}
	}
}

func (m *jasnaStreamManager) Open(ctx context.Context, filePath string, extraArgs []string, startTime float64) error {
	m.mu.Lock()
	if err := m.ensureRunningLocked(extraArgs); err != nil {
		m.mu.Unlock()
		return err
	}
	sameFile := m.currentFile == filePath
	m.lastActivity = time.Now()
	m.mu.Unlock()

	if sameFile && startTime == 0 {
		logger.Debugf("[jasna] stream already open for %s, skipping /open", filePath)
		return nil
	}

	payload := map[string]interface{}{"path": filePath}
	if startTime > 0 {
		payload["start"] = startTime
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, jasnaBaseURL+"/open", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("jasna open request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("jasna open: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jasna open returned %d: %s", resp.StatusCode, string(respBody))
	}

	m.mu.Lock()
	m.currentFile = filePath
	m.mu.Unlock()

	logger.Infof("[jasna] open response: %s", string(respBody))
	return nil
}

func (m *jasnaStreamManager) callStop() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, jasnaBaseURL+"/open", nil)
	if err != nil {
		return
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func (m *jasnaStreamManager) StreamURL() string {
	return jasnaBaseURL + "/stream.m3u8"
}

func (m *jasnaStreamManager) touchActivity() {
	m.mu.Lock()
	m.lastActivity = time.Now()
	m.mu.Unlock()
}

func (m *jasnaStreamManager) idleMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.parentCtx.Done():
			return
		case <-ticker.C:
			m.mu.Lock()
			if m.cmd != nil && time.Since(m.lastActivity) > jasnaIdleTimeout {
				logger.Infof("[jasna] idle timeout reached, stopping process")
				m.callStop()
				m.stopProcessLocked()
			}
			m.mu.Unlock()
		}
	}
}
