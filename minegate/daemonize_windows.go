package minegate

import (
	log "github.com/jackyyf/golog"
)

func Daemonize(_ ...interface{}) error {
	// Just a stub, does not do anything special.
	log.Warn("daemonize is not supported on windows!")
	return nil
}
