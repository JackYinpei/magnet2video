//go:build !linux && !darwin

// Fallback for any platform we don't ship to (Windows, etc.). Returns zeros so
// the heartbeat keeps working without a syscall dependency.
package heartbeat

func (p *Publisher) diskInfoGB() (int64, int64) {
	return 0, 0
}
