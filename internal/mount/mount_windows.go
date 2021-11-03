//go:build windows

package mount

import (
	"golang.org/x/sys/windows"
)

func mounts() ([]Mount, error) {
}

type windowsWatcher struct {
}
