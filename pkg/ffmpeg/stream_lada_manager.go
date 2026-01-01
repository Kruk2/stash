package ffmpeg

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	stashExec "github.com/stashapp/stash/pkg/exec"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/logger"
)

type ladaStreamManager struct {
	lockManager *fsutil.ReadLockManager
	context     context.Context

	mu     sync.Mutex
	cancel context.CancelFunc

	cmd   *exec.Cmd
	stdin io.WriteCloser

	done chan struct{}
}

func newLadaStreamManager(lockManager *fsutil.ReadLockManager, ctx context.Context) *ladaStreamManager {
	return &ladaStreamManager{
		lockManager: lockManager,
		context:     ctx,
	}
}

func (m *ladaStreamManager) Shutdown() {
	m.mu.Lock()
	m.stopLocked()
	m.mu.Unlock()
}

func (m *ladaStreamManager) stopLocked() {
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
	m.stdin = nil
}

func (m *ladaStreamManager) ensureRunningLocked() error {
	if m.cmd != nil {
		select {
		case <-m.done:
			m.stopLocked()
		default:
			return nil
		}
	}

	m.stopLocked()

	procCtx, cancel := context.WithCancel(m.context)
	cmd := stashExec.CommandContext(procCtx, resolveLadaStreamerPath())

	cmd.Stdout = io.Discard
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return err
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return err
	}

	done := make(chan struct{})

	m.cancel = cancel
	m.cmd = cmd
	m.stdin = stdin
	m.done = done

	go m.consumeStderr(stderr, done)
	go m.wait(cmd, done)

	return nil
}

func (m *ladaStreamManager) wait(cmd *exec.Cmd, done chan struct{}) {
	err := cmd.Wait()

	// ignore EPIPE/ECONNRESET/cancel-like errors; the process may be killed on stream switch
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) && !errors.Is(err, context.Canceled) {
		logger.Errorf("[lada] process exited: %v", err)
	}

	close(done)
}

func (m *ladaStreamManager) consumeStderr(stderr io.Reader, done <-chan struct{}) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			logger.Errorf("[lada] %s", line)
		}
		select {
		case <-done:
			return
		default:
		}
	}
}

func (m *ladaStreamManager) sendSeekPipeLocked(startTime float64, inputPath string, pipePath string) error {
	if m.stdin == nil {
		return fmt.Errorf("lada stdin not available")
	}

	seconds := strconv.FormatFloat(startTime, 'f', -1, 64)
	_, err := io.WriteString(m.stdin, "SEEK "+seconds+" "+inputPath+" "+pipePath+"\n")
	return err
}

func (m *ladaStreamManager) Serve(w http.ResponseWriter, r *http.Request, inputPath string, startTime float64) error {
	return m.servePipe(w, r, inputPath, startTime)
}

func (m *ladaStreamManager) servePipe(w http.ResponseWriter, r *http.Request, inputPath string, startTime float64) error {
	pipeName := "stash-lada-" + uuid.NewString()
	pipePath := `\\.\pipe\` + pipeName

	lockCtx := m.lockManager.ReadLock(r.Context(), inputPath)
	defer lockCtx.Cancel()

	ln, err := listenLadaPipe(pipePath)
	if err != nil {
		return err
	}
	defer ln.Close()

	acceptCh := make(chan net.Conn, 1)
	acceptErrCh := make(chan error, 1)

	go func() {
		c, err := ln.Accept()
		if err != nil {
			acceptErrCh <- err
			return
		}
		acceptCh <- c
	}()

	go func() {
		<-r.Context().Done()
		_ = ln.Close()
	}()

	m.mu.Lock()
	if err := m.ensureRunningLocked(); err != nil {
		m.mu.Unlock()
		return err
	}

	if err := m.sendSeekPipeLocked(startTime, inputPath, pipePath); err != nil {
		m.mu.Unlock()
		return err
	}
	m.mu.Unlock()

	var conn net.Conn
	select {
	case conn = <-acceptCh:
	case err := <-acceptErrCh:
		return err
	case <-r.Context().Done():
		return r.Context().Err()
	}
	defer conn.Close()

	go func() {
		<-r.Context().Done()
		_ = conn.Close()
	}()

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", MimeWebmVideo)
	w.WriteHeader(http.StatusOK)

	_, err = io.Copy(w, conn)
	if err != nil && !errors.Is(err, syscall.EPIPE) && !errors.Is(err, syscall.ECONNRESET) {
		logger.Errorf("[lada] error serving streamed video file: %v", err)
	}

	w.(http.Flusher).Flush()

	return nil
}
