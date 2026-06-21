//go:build windows

package skillstore

import "os"

// flock 在 Windows 上退化为 no-op。Skill Box v1 在桌面端是单进程实例,
// 进程内锁(inprocLocks)已经能保证并发安全。跨进程并发由 OS 文件系统
// 自身保证最终一致性(目录 rename 是原子的)。
func flock(f *os.File) error { return nil }
func funlock(f *os.File) error { return nil }
