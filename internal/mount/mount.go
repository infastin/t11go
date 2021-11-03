package mount

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

type Mount struct {
	Device string
	Mpoint string
	Ftype  string
	Bsize  int64
	Blocks uint64
	Bavail uint64
}

func Mounts(fd int) ([]Mount, error) {
	mnts := os.NewFile(uintptr(fd), "/proc/mounts")

	mounts := []Mount(nil)
	reader := bufio.NewReader(mnts)

	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}

		if err != nil {
			return mounts, err
		}

		if line[0] == '/' {
			str := string(line)
			fields := strings.Split(str, " ")
			mntPoint := fields[1]

			if mntPoint != "/" && !strings.HasPrefix(mntPoint, "/mnt") &&
				!strings.HasPrefix(mntPoint, "/media") &&
				!strings.HasPrefix(mntPoint, "/run/media") &&
				!strings.HasPrefix(mntPoint, "/run/mount") {
				continue
			}

			var statfs unix.Statfs_t
			unix.Statfs(fields[1], &statfs)

			mounts = append(mounts, Mount{
				Device: fields[0],
				Mpoint: fields[1],
				Ftype:  fields[2],
				Bsize:  statfs.Bsize,
				Blocks: statfs.Blocks,
				Bavail: statfs.Bavail,
			})
		}
	}

	return mounts, nil
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

type Watcher struct {
	mounts []Mount
	Events chan struct{}
	Errors chan error
}

func NewWatcher() (*Watcher, error) {
	fd, err := unix.Open("/proc/mounts", unix.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer unix.Close(fd)

	mounts, err := Mounts(fd)
	if err != nil {
		return nil, err
	}

	return &Watcher{
		mounts: mounts,
		Events: make(chan struct{}),
		Errors: make(chan error),
	}, nil
}

func (w *Watcher) Mounts() []Mount {
	return w.mounts
}

func (w *Watcher) Watch() error {
	fd, err := unix.Open("/proc/mounts", unix.O_RDONLY, 0)
	if err != nil {
		return err
	}

	var edfs unix.FdSet
	edfs.Zero()
	edfs.Set(fd)

	go func() {
		defer w.Close()
		defer unix.Close(fd)

		for {
			n, err := unix.Select(fd+1, nil, nil, &edfs, nil)
			if n < 0 {
				if err == unix.EINTR {
					continue
				}

				w.Errors <- err
				return
			}

			if edfs.IsSet(fd) {
				unix.Seek(fd, 0, unix.SEEK_SET)
				mounts, err := Mounts(fd)
				if err != nil {
					w.Errors <- err
					return
				}

				w.mounts = mounts
				w.Events <- struct{}{}

				edfs.Zero()
				edfs.Set(fd)
			}
		}
	}()

	return nil
}

func (w *Watcher) Close() {
	close(w.Errors)
	close(w.Events)
}
