//go:build !windows

package ffmpeg

import (
	"fmt"
	"net"
)

func listenLadaPipe(pipePath string) (net.Listener, error) {
	return nil, fmt.Errorf("lada streaming via named pipes is only supported on windows (attempted %q)", pipePath)
}

