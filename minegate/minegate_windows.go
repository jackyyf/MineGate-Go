package minegate

import (
	"runtime"
)

func Run() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	PreLoadConfig()
	confInit()
	PostLoadConfig()
	ServerSocket()
}
