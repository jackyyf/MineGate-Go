package minegate

import (
	"bufio"
	"fmt"
	log "github.com/jackyyf/golog"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type ReaderWriter interface {
	io.ByteReader
	io.Writer
}

type WrapedSocket struct {
	sock       net.Conn
	id         uint64
	log_prefix string
	client     bool
	*bufio.Reader
}

var counter uint64 = 0

func WrapUpstreamSocket(conn net.Conn, cws *WrapedSocket) (ws *WrapedSocket) {
	ws = new(WrapedSocket)
	ws.sock = conn
	ws.Reader = bufio.NewReader(conn)
	ws.id = cws.Id()
	ws.log_prefix = fmt.Sprintf("[#%d %s] ", ws.id, conn.RemoteAddr())
	ws.client = false
	return
}

func WrapClientSocket(conn net.Conn) (ws *WrapedSocket) {
	ws = new(WrapedSocket)
	ws.sock = conn
	ws.Reader = bufio.NewReader(conn)
	ws.id = counter
	counter++
	ws.log_prefix = fmt.Sprintf("[#%d %s] ", ws.id, conn.RemoteAddr())
	ws.client = true
	return
}

func (ws *WrapedSocket) Id() uint64 {
	return ws.id
}

func (ws *WrapedSocket) Write(b []byte) (n int, err error) {
	return ws.sock.Write(b)
}

func (ws *WrapedSocket) Close() error {
	if !ws.client {
		// Disconnect from upstream.
		de := new(DisconnectEvent)
		de.connID = ws.id
		de.RemoteAddr = ws.sock.RemoteAddr().(*net.TCPAddr)
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

func ToBool(val interface{}) (res bool) {
	// First check if val is nil
	if val == nil {
		return false
	}
	// Next try convert to bool
	switch val := val.(type) {
	case bool:
		return val
	case int:
		return val > 0
	case int64:
		return val > 0
	case uint, uint64:
		return val != 0
	case string:
		val = strings.ToLower(val)
		return val == "y" || val == "yes" || val == "true" || val == "on"
	default:
		return true
	}
}

func ToInt(val interface{}) (res int64) {
	// nil is 0
	if val == nil {
		return 0
	}
	switch val := val.(type) {
	case bool:
		if val {
			return 1
		} else {
			return 0
		}
	case int:
		return int64(val)
	case int64:
		return int64(val)
	case uint:
		return int64(val)
	case uint64:
		return int64(val)
	case float32:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		res, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			return res
		} else {
			return 0
		}
	default:
		return 0
	}
}

func ToUint(val interface{}) (res uint64) {
	// nil is 0
	if val == nil {
		return 0
	}
	switch val := val.(type) {
	case bool:
		if val {
			return 1
		} else {
			return 0
		}
	case int:
		return uint64(val)
	case int64:
		return uint64(val)
	case uint:
		return uint64(val)
	case uint64:
		return uint64(val)
	case float32:
		return uint64(val)
	case float64:
		return uint64(val)
	case string:
		res, err := strconv.ParseUint(val, 0, 64)
		if err != nil {
			return res
		} else {
			return 0
		}
	default:
		return 0
	}
}

func ToString(val interface{}) (res string) {
	if val == nil {
		return ""
	}
	switch val := val.(type) {
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
