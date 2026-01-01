package ffmpeg

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
)

func ladaStreamerFilename() string {
	if runtime.GOOS == "windows" {
		return "lada-streamer.exe"
	}
	return "lada-streamer"
}

func resolveLadaStreamerPath() string {
	if p := os.Getenv("LADA_STREAMER_PATH"); p != "" {
		return p
	}

	name := ladaStreamerFilename()
	if p, err := exec.LookPath(name); err == nil {
		return p
	}

	exe, err := os.Executable()
	if err != nil {
		return name
	}

	candidate := filepath.Join(filepath.Dir(exe), name)
	if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
		return candidate
	}

	return name
}

func (sm *StreamManager) ServeLada(w http.ResponseWriter, r *http.Request, vf *models.VideoFile, startTime float64) {
	if sm.lada == nil {
		http.Error(w, "Lada streamer not available", http.StatusServiceUnavailable)
		return
	}
	if err := sm.lada.Serve(w, r, vf.Path, startTime); err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.Errorf("[lada] error streaming video file: %v", err)
		}
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(err.Error())); err != nil {
			logger.Warnf("[lada] error writing response: %v", err)
		}
	}
}
