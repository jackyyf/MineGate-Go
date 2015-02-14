// +build !windows

package main

import (
	"fmt"
	"os"
	"syscall"
)

func Daemonize() (e error) {
	// var ret, err uintptr
	ret, _, err := syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	if err != 0 {
		return fmt.Errorf("fork error: %d", err)
	}
	if ret != 0 {
		// Parent
		os.Exit(0)
	}

	// We are now child

	if err, _ := syscall.Setsid(); err < 0 {
		return fmt.Errorf("setsid error: %d", err)
	}

	os.Chdir("/")
	f, e := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if e != nil {
		return
	}
	fd := int(f.Fd())
	syscall.Dup2(fd, int(os.Stdin.Fd()))
	syscall.Dup2(fd, int(os.Stdout.Fd()))
	syscall.Dup2(fd, int(os.Stderr.Fd()))

	return nil
}
