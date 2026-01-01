//go:build windows

package ffmpeg

import (
	"net"

	"github.com/Microsoft/go-winio"
)

func listenLadaPipe(pipePath string) (net.Listener, error) {
	return winio.ListenPipe(pipePath, nil)
}

