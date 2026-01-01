package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
)

func jasnaCliFilename() string {
	if runtime.GOOS == "windows" {
		return "jasna-cli.exe"
	}
	return "jasna-cli"
}

func resolveJasnaCliCommand() (string, []string) {
	if p := os.Getenv("JASNA_CLI_PATH"); p != "" {
		parts := strings.Fields(p)
		return parts[0], parts[1:]
	}

	name := jasnaCliFilename()
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return name, nil
	}

	candidate := filepath.Join(filepath.Dir(exe), name)
	if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
		return candidate, nil
	}

	return name, nil
}

func (sm *StreamManager) ServeJasna(w http.ResponseWriter, r *http.Request, vf *models.VideoFile, extraArgs []string, startTime float64) {
	if sm.jasna == nil {
		http.Error(w, "Jasna streamer not available", http.StatusServiceUnavailable)
		return
	}

	if err := sm.jasna.Open(r.Context(), vf.Path, extraArgs, startTime); err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.Errorf("[jasna] error opening file for streaming: %v", err)
		}
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	go func() {
		<-r.Context().Done()
		sm.jasna.touchActivity()
	}()

	streamURL := sm.jasna.StreamURL()
	if startTime > 0 {
		streamURL = fmt.Sprintf("%s?start=%.3f", streamURL, startTime)
	}
	http.Redirect(w, r, streamURL, http.StatusFound)
}
