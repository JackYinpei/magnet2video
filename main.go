// Package main provides the main entry point for the application.
// Author: Done-0
// Created: 2025-09-25
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"magnet2video/cmd"
	"magnet2video/configs"
)

func main() {
	modeFlag := flag.String("mode", "", "deployment mode: all | server | worker (default: value of APP.MODE in config, or 'all')")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-mode=all|server|worker]\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nmode=server: run API + DB + event sink (no download/transcode workers)")
		fmt.Fprintln(os.Stderr, "mode=worker: run download + transcode + cloud-upload workers (no DB/Gin)")
		fmt.Fprintln(os.Stderr, "mode=all:    single-process, all components in one binary")
	}
	flag.Parse()

	mode := strings.ToLower(strings.TrimSpace(*modeFlag))
	switch mode {
	case "", configs.ModeAll, configs.ModeServer, configs.ModeWorker:
		// valid (empty means defer to config)
	default:
		fmt.Fprintf(os.Stderr, "unknown mode: %q\n", mode)
		os.Exit(2)
	}

	cmd.RunMode(mode)
}
