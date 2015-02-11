package main

import (
	"os"
	"os/signal"
	"syscall"
)

func main() {
	LogInit()
	ConfInit()
	go ServerSocket()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP)
	for {
		cur := <-sig
		if cur != syscall.SIGHUP {
			Errorf("Trapped unexpected signal: %s", cur.String())
			continue
		}
		Warn("SIGHUP caught, reloading config...")
		ConfReload()
	}
}
