//go:build windows

package mount

import (
	"golang.org/x/sys/windows"
	"syscall"
	"unsafe"
)

const (
	errnoERROR_IO_PENDING = 997
)

var (
	errERROR_IO_PENDING error = syscall.Errno(errnoERROR_IO_PENDING)
	errERROR_EINVAL     error = syscall.EINVAL
)

func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return errERROR_EINVAL
	case errnoERROR_IO_PENDING:
		return errERROR_IO_PENDING
	}
	return e
}

var (
	modkernel32 = windows.MustLoadDLL("kernel32.dll")
)

var (
	procGetDiskFreeSpaceW = modkernel32.MustFindProc("GetDiskFreeSpaceW")
)

func getDiskFreeSpace(rootPathName *uint16, sectorsPerCluster *uint32, bytesPerSector *uint32, numberOfFreeClusters *uint32, totalNumberOfClusters *uint32) (err error) {
	r1, _, e1 := syscall.Syscall6(procGetDiskFreeSpaceW.Addr(), 5, uintptr(unsafe.Pointer(rootPathName)),
		uintptr(unsafe.Pointer(sectorsPerCluster)),
		uintptr(unsafe.Pointer(bytesPerSector)),
		uintptr(unsafe.Pointer(numberOfFreeClusters)),
		uintptr(unsafe.Pointer(totalNumberOfClusters)),
		0)
	if r1 == 0 {
		err = errnoErr(e1)
	}
	return
}

func mounts() ([]Mount, error) {
	var mounts []Mount
	var volumeName [windows.MAX_PATH]uint16

	handle, err := windows.FindFirstVolume((*uint16)(unsafe.Pointer(&volumeName)), windows.MAX_PATH)
	if err != nil {
		return nil, err
	}
	defer windows.FindVolumeClose(handle)

	for {
		volName := syscall.UTF16ToString(volumeName[:])
		index := len(volName) - 1
		if volName[0] != '\\' ||
			volName[1] != '\\' ||
			volName[2] != '?' ||
			volName[3] != '\\' ||
			volName[index] != '\\' {
			err = windows.ERROR_BAD_DEVICE_PATH
			return nil, err
		}

		var deviceName [windows.MAX_PATH]uint16
		volumeName[index] = 0
		_, err := windows.QueryDosDevice((*uint16)(unsafe.Pointer(&volumeName[4])),
			(*uint16)(unsafe.Pointer(&deviceName)), windows.MAX_PATH)
		if err != nil {
			return nil, err
		}
		volumeName[index] = '\\'

		var volumePathName [windows.MAX_PATH]uint16
		err = windows.GetVolumePathNamesForVolumeName((*uint16)(unsafe.Pointer(&volumeName)),
			(*uint16)(unsafe.Pointer(&volumePathName)), windows.MAX_PATH, nil)
		if err != nil {
			return nil, err
		}

		if volumePathName[0] != 0 {
			var lpFileSystemNameBuffer [windows.MAX_PATH]uint16
			err = windows.GetVolumeInformation((*uint16)(unsafe.Pointer(&volumePathName)), nil, 0,
				nil, nil, nil,
				(*uint16)(unsafe.Pointer(&lpFileSystemNameBuffer)), windows.MAX_PATH)
			if err != nil {
				return nil, err
			}

			var sectorsPerCluster, bytesPerSector, numberOfFreeClusters, totalNumberOfClusters uint32
			err = getDiskFreeSpace((*uint16)(unsafe.Pointer(&volumePathName)), &sectorsPerCluster,
				&bytesPerSector, &numberOfFreeClusters, &totalNumberOfClusters)
			if err != nil {
				return nil, err
			}

			mounts = append(mounts, Mount{
				Device: syscall.UTF16ToString(deviceName[:]),
				Mpoint: syscall.UTF16ToString(volumePathName[:]),
				Ftype:  syscall.UTF16ToString(lpFileSystemNameBuffer[:]),
				Bsize:  uint64(sectorsPerCluster) * uint64(bytesPerSector),
				Blocks: uint64(totalNumberOfClusters),
				Bavail: uint64(numberOfFreeClusters),
			})
		}

		err = windows.FindNextVolume(handle, (*uint16)(unsafe.Pointer(&volumeName)), windows.MAX_PATH)
		if err != nil {
			if err != windows.ERROR_NO_MORE_FILES {
				return nil, err
			}

			break
		}
	}

	return mounts, nil
}

type windowsWatcher struct {
	mounts []Mount
	events chan struct{}
	errors chan error
}

func newWatcher() (Watcher, error) {
	mounts, err := mounts()
	if err != nil {
		return nil, err
	}

	return &windowsWatcher{
		mounts: mounts,
		events: make(chan struct{}),
		errors: make(chan error),
	}, nil
}

func (w *windowsWatcher) Mounts() []Mount {
	return w.mounts
}

func (w *windowsWatcher) Watch() error {
	return ErrWatchUnsupported
}

func (w *windowsWatcher) Events() chan struct{} {
	return w.events
}

func (w *windowsWatcher) Errors() chan error {
	return w.errors
}
