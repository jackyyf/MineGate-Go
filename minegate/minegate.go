package main

import (
	log "github.com/jackyyf/golog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.SetLogLevel(log.DEBUG)
	// ConfInit()
	go ServerSocket()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGUSR1)
	for {
		cur := <-sig
		switch cur {
		case syscall.SIGHUP:
			log.Warn("SIGHUP caught, reloading config...")
			ConfReload()
		case syscall.SIGUSR1:
			log.Warn("SIGUSR1 caught, rotating log...")
			log.Rotate()
		default:
			log.Errorf("Trapped unexpected signal: %s", cur.String())
			continue
		}
	}
}
