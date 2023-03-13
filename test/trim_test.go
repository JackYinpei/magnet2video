package test

import (
	"fmt"
	"path"
	"peer2http/util"
	"strings"
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

func TestTrimName(t *testing.T) {
	filename := "ubuntu-20.04.5-live-server-amd64.iso.torrent"
	fmt.Println("qian mian", filename)
	extend := path.Ext(filename)
	fmt.Println(extend)
	nameonly := strings.TrimSuffix(filename, extend)
	fmt.Println(nameonly)
}
