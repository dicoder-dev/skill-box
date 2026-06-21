//go:build unix

package skillstore

import (
	"os"
	"syscall"
)

// flock 对文件描述符加排他锁。文件关闭时锁自动释放。
func flock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
}

// funlock 释放锁。
func funlock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
