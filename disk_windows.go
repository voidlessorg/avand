//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

func diskFree(path string) (free, total int64) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0
	}
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")
	var fa, tb, tf uint64
	r, _, _ := proc.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&fa)), uintptr(unsafe.Pointer(&tb)), uintptr(unsafe.Pointer(&tf)))
	if r == 0 {
		return 0, 0
	}
	return int64(fa), int64(tb)
}
