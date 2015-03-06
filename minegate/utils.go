package minegate

import (
	"bufio"
	"fmt"
	log "github.com/jackyyf/golog"
	"io"
	"net"
	"time"
)

type ReaderWriter interface {
	io.ByteReader
	io.Writer
}

type WrapedSocket struct {
	sock       net.Conn
	id         uintptr
	log_prefix string
	client     bool
	*bufio.Reader
}

var counter uintptr = 0

func WrapUpstreamSocket(conn net.Conn, cws *WrapedSocket) (ws *WrapedSocket) {
	ws = new(WrapedSocket)
	ws.sock = conn
	ws.Reader = bufio.NewReader(conn)
	ws.id = cws.Id()
	ws.log_prefix = fmt.Sprintf("[#%d %s]", ws.id, conn.RemoteAddr())
	ws.client = false
	return
}

func WrapClientSocket(conn net.Conn) (ws *WrapedSocket) {
	ws = new(WrapedSocket)
	ws.sock = conn
	ws.Reader = bufio.NewReader(conn)
	ws.id = counter
	counter++
	ws.log_prefix = fmt.Sprintf("[#%d %s]", ws.id, conn.RemoteAddr())
	ws.client = true
	return
}

func (ws *WrapedSocket) Id() uintptr {
	return ws.id
}

func (ws *WrapedSocket) Write(b []byte) (n int, err error) {
	return ws.sock.Write(b)
}

func (ws *WrapedSocket) Close() error {
	if ws.client {
		de := new(DisconnectEvent)
		de.connID = ws.id
		de.RemoteAddr = ws.sock.RemoteAddr()
		Disconnect(de)
	}
	return ws.sock.Close()
}

func (ws *WrapedSocket) LocalAddr() net.Addr {
	return ws.sock.LocalAddr()
}

func (ws *WrapedSocket) RemoteAddr() net.Addr {
	return ws.sock.RemoteAddr()
}

func (ws *WrapedSocket) SetDeadline(t time.Time) error {
	return ws.sock.SetDeadline(t)
}

func (ws *WrapedSocket) SetReadDeadline(t time.Time) error {
	return ws.sock.SetReadDeadline(t)
}

func (ws *WrapedSocket) SetWriteDeadline(t time.Time) error {
	return ws.sock.SetWriteDeadline(t)
}

func (ws *WrapedSocket) SetTimeout(d time.Duration) error {
	return ws.sock.SetDeadline(time.Now().Add(d))
}

func (ws *WrapedSocket) SetReadTimeout(d time.Duration) error {
	return ws.sock.SetReadDeadline(time.Now().Add(d))
}

func (ws *WrapedSocket) SetWriteTimeout(d time.Duration) error {
	return ws.sock.SetReadDeadline(time.Now().Add(d))
}

func (ws *WrapedSocket) Debugf(format string, v ...interface{}) {
	log.Debugf(ws.log_prefix+format, v...)
}

func (ws *WrapedSocket) Infof(format string, v ...interface{}) {
	log.Infof(ws.log_prefix+format, v...)
}

func (ws *WrapedSocket) Warnf(format string, v ...interface{}) {
	log.Warnf(ws.log_prefix+format, v...)
}

func (ws *WrapedSocket) Errorf(format string, v ...interface{}) {
	log.Errorf(ws.log_prefix+format, v...)
}

func (ws *WrapedSocket) Fatalf(format string, v ...interface{}) {
	log.Fatalf(ws.log_prefix+format, v...)
}
