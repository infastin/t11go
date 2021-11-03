package mount

import (
	"fmt"
)

type Mount struct {
	Device string
	Mpoint string
	Ftype  string
	Bsize  int64
	Blocks uint64
	Bavail uint64
}

type Watcher interface {
	Mounts() []Mount
	Watch() error
	Events() chan struct{}
	Errors() chan error
}

func NewWatcher() (Watcher, error) {
	return newWatcher()
}

func humanReadable(bytes uint64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}

	fbytes := float64(bytes)
	fbytes /= 1024

	if fbytes < 1024 {
		return fmt.Sprintf("%.2f KiB", fbytes)
	}
	fbytes /= 1024

	if fbytes < 1024 {
		return fmt.Sprintf("%.2f MiB", fbytes)
	}
	fbytes /= 1024

	if fbytes < 1024 {
		return fmt.Sprintf("%.2f GiB", fbytes)
	}
	fbytes /= 1024

	return fmt.Sprintf("%.2f TiB", fbytes)
}

func (m Mount) Size() string {
	bytes := m.Blocks * uint64(m.Bsize)
	return humanReadable(bytes)
}

func (m Mount) Avail() string {
	bytes := m.Bavail * uint64(m.Bsize)
	return humanReadable(bytes)
}
