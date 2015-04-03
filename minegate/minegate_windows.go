package minegate

import (
	"runtime"
	log "github.com/jackyyf/golog"
)

func Run() {
	PreLoadConfig()
	confInit()
	PostLoadConfig()
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Infof("MineGate %s started.", version_full)
	ServerSocket()
}
