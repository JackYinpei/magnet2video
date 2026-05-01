// Darwin disk-stat implementation. macOS's syscall.Statfs_t has no Frsize
// field; Bsize is the right choice on native APFS/HFS+ mounts.
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
	const gb = 1024 * 1024 * 1024
	free := int64(stat.Bavail) * int64(stat.Bsize) / gb
	total := int64(stat.Blocks) * int64(stat.Bsize) / gb
	return free, total
}
