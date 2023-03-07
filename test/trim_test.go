package test

import (
	"fmt"
	"peer2http/util"
	"testing"
)

func TestTrim(t *testing.T) {
	tracker := util.NewTracker("../tracker.txt")
	trackers := tracker.GetTrackerList()
	for _, t := range trackers {
		fmt.Println(t)
	}
}
