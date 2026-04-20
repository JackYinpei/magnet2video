// Package wire: helpers for worker id generation.
// Author: magnet2video
// Created: 2026-04-20
package wire

import (
	"fmt"
	"os"
	"time"
)

// defaultWorkerID builds a fallback worker id from hostname + pid + timestamp.
func defaultWorkerID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "worker"
	}
	return fmt.Sprintf("%s-%d-%d", host, os.Getpid(), time.Now().Unix()%100000)
}
