// Linux disk-stat implementation. Uses Frsize (the POSIX "fundamental block
// size") rather than Bsize. On most Linux filesystems they are identical, but
// virtio-fs / 9p mounts (Docker Desktop on macOS) report Bsize=1MiB while the
// block counts are still in 4KiB units — using Bsize there inflates the
// reported size by ~256x. Frsize is consistent with what `df` shows.
package heartbeat

import (
	"os"
	"syscall"
)

func (p *Publisher) diskInfoGB() (int64, int64) {
	if p.downloadDir == "" {
		return 0, 0
	}
	path := p.downloadDir
	if _, err := os.Stat(path); err != nil {
		return 0, 0
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0
	}
	bs := int64(stat.Frsize)
	if bs <= 0 {
		bs = int64(stat.Bsize)
	}
	const gb = 1024 * 1024 * 1024
	free := int64(stat.Bavail) * bs / gb
	total := int64(stat.Blocks) * bs / gb
	return free, total
}
