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

func TestHttpGet(t *testing.T) {
	downloader := util.NewDownloader("C:\\goproj\\peer2HttpDemo\\torrents")
	downloader.SetMagnet("magnet:?xt=urn:btih:DD5B2337F90EE4D34012F0C270825B9EFF6A7960")
	fileName := downloader.GetTorrent()
	if fileName == "" {
		t.Failed()
	}
}
