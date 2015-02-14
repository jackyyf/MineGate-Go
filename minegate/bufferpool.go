package main

import (
	log "github.com/jackyyf/golog"
)

type BufferPool struct {
	chunk_size int
	pool       chan []byte
}

type BufferQueue chan []byte

var pool *BufferPool

func InitPool(num, size int) {
	log.Debugf("memory pool buffer size: %d * %d bytes", num, size)
	pool = &BufferPool{
		chunk_size: size,
		pool:       make(chan []byte, num),
	}
}

func Allocate() (buff []byte) {
	select {
	case buff = <-pool.pool:
	default:
		log.Info("No available memory, allocating new.")
		buff = make([]byte, pool.chunk_size)
	}
	return
}

func Free(buff []byte) {
	if len(buff) != pool.chunk_size {
		buff = buff[0:pool.chunk_size]
	}
	select {
	case pool.pool <- buff:
	default:
		log.Info("Memory pool full, discarding")
	}
	return
}

func NewBufferQueue(len int) BufferQueue {
	log.Debugf("bufferqueue, size=%d", len)
	return make(chan []byte, len)
}
