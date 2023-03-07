package util

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Trackers struct {
	trackers    [][]string
	trackerFile string
}

func NewTracker(trackerFilePath string) *Trackers {
	tracker := &Trackers{
		trackers:    nil,
		trackerFile: trackerFilePath,
	}
	return tracker
}

func (t *Trackers) GetTrackerList() [][]string {
	file, err := os.Open(t.trackerFile)
	if err != nil {
		fmt.Println("\n", err)
		panic("打开Tracker file 文件失败")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	trackerLine := make([]string, 0)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.Trim(line, "\n")
		if len(line) == 0 {
			continue
		} else {
			trackerLine = append(trackerLine, line)
		}
	}
	for _, tracker := range trackerLine {
		t.trackers = append(t.trackers, []string{tracker})
		//t.trackers[i] = []string{tracker}
	}
	return t.trackers
}
