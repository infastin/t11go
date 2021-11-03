//go:build linux

package mount

import (
	"bufio"
	"io"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

func mounts(fd int) ([]Mount, error) {
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

type linuxWatcher struct {
	mounts []Mount
	events chan struct{}
	errors chan error
}

func newWatcher() (Watcher, error) {
	fd, err := unix.Open("/proc/mounts", unix.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer unix.Close(fd)

	mounts, err := mounts(fd)
	if err != nil {
		return nil, err
	}

	return &linuxWatcher{
		mounts: mounts,
		events: make(chan struct{}),
		errors: make(chan error),
	}, nil
}

func (w *linuxWatcher) Mounts() []Mount {
	return w.mounts
}

func (w *linuxWatcher) Events() chan struct{} {
	return w.events
}

func (w *linuxWatcher) Errors() chan error {
	return w.errors
}

func (w *linuxWatcher) Watch() error {
	fd, err := unix.Open("/proc/mounts", unix.O_RDONLY, 0)
	if err != nil {
		return err
	}

	var edfs unix.FdSet
	edfs.Zero()
	edfs.Set(fd)

	go func() {
		defer w.close()

		for {
			n, err := unix.Select(fd+1, nil, nil, &edfs, nil)
			if n < 0 {
				if err == unix.EINTR {
					continue
				}

				w.errors <- err
				return
			}

			if edfs.IsSet(fd) {
				unix.Seek(fd, 0, unix.SEEK_SET)
				mounts, err := mounts(fd)
				if err != nil {
					w.errors <- err
					return
				}

				w.mounts = mounts
				w.events <- struct{}{}

				edfs.Zero()
				edfs.Set(fd)
			}
		}
	}()

	return nil
}

func (w *linuxWatcher) close() {
	close(w.errors)
	close(w.events)
}
