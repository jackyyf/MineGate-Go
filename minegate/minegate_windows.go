package minegate

import (
	"runtime"
)

func Run() {
	PreLoadConfig()
	confInit()
	PostLoadConfig()
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Infof("MineGate %s started.", version_full)
	ServerSocket()
}
