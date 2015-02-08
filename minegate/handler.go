package main

import (
	"net"
	"encoding/binary"
	"unicode/utf16"
)

var total_online uint32

type Handshake struct {
	version int
	host string
	port uint16
	isPing bool
}


func handlePing(conn net.Conn, upstream *Upstream) {

}

func ReadTo(conn net.Conn, queue BufferQueue, notifyqueue chan byte) {
	for {
		buff := Allocate()
		n, err := conn.Read(buff)
		if n > 0 {
			select {
			case <- notifyqueue:
				close(queue)
				Info("recv side signaled, closing.")
				return
			case queue <- buff[:n]:
				Debugf("recv %d bytes", n)
			}
			continue
		}
		if err != nil {
			Warnf("recv error: %s", err.Error())
			conn.Close()
			notifyqueue <- '\x00'
			return
		}
	}
}

func WriteTo(conn net.Conn, queue BufferQueue, notifyqueue chan byte) {
	for {
		select {
		case buff := <- queue:
			_, err := conn.Write(buff)
			Free(buff)
			if err != nil {
				Warnf("send error: %s", err.Error())
				conn.Close()
				select {
				case <- notifyqueue:
					close(queue)
					return
				default:
				}
				notifyqueue <- '\x01'
				return
			}
			continue
		case <- notifyqueue:
			Info("send side signaled, closing.")
			close(queue)
			return
		}
	}
}

func startProxy(conn net.Conn, upstream *Upstream, initial []byte) {
	if _, _, err := net.SplitHostPort(upstream.Server); err != nil {
		upstream.Server += ":25565";
	}
	upconn, err := net.Dial("tcp", upstream.Server)
	if err != nil {
		Errorf("Unable to connect to upstream %s", upstream.Server)
		KickClient(conn, "502 Bad Gateway.")
		conn.Close()
		return
	}
	c2squeue := NewBufferQueue(4)
	s2cqueue := NewBufferQueue(32)
	c2sstatus := make(chan byte, 1)
	s2cstatus := make(chan byte, 1)
	c2squeue <- initial
	go ReadTo(conn, c2squeue, c2sstatus)
	go WriteTo(upconn, c2squeue, c2sstatus)
	go ReadTo(upconn, s2cqueue, s2cstatus)
	go WriteTo(conn, s2cqueue, s2cstatus)
}

func ServerSocket() {
	s, err := net.Listen("tcp", config.Listen_addr)
	if err != nil {
		Fatalf("error listening on %s: %s", config.Listen_addr, err.Error())
	}
	Infof("Server listened on %s", config.Listen_addr)
	total_online = 0
	// 4 MB pool
	InitPool(1024, 4096)
	for {
		conn, err := s.Accept()
		if err != nil {
			Warnf("listen_socket: error when accepting: %s", err.Error())
			continue
		}
		Infof("listen_socket: new client %s", conn.RemoteAddr())
		go ClientSocket(conn)
	}
}

func ClientSocket(conn net.Conn) {
	buff := Allocate()
	orig := buff
	n, err := conn.Read(buff)
	if n == 0 {
		Warnf("no data received, disconnecting %s", conn.RemoteAddr())
		Free(buff)
		conn.Close()
		return
	}
	if err != nil {
		Warnf("recv error from %s: %s", conn.RemoteAddr(), err.Error())
		Free(buff)
		conn.Close()
		return
	}
	if buff[0] == 0xFE || buff[0] == 0x02 {
		Warnf("%s: using old (1.6-) protocol, disconnecting", conn.RemoteAddr())
		// 1.6- protocol, disconnect them.
		msg := []rune("outdated client. minegate requires 1.7+")
		msglen := uint16(len(msg))
		conn.Write([]byte{'\xFF'})
		binary.Write(conn, binary.BigEndian, msglen)
		binary.Write(conn, binary.BigEndian, utf16.Encode(msg))
		conn.Close()
		Free(buff)
		return
	}
	pktlen, lenlen := binary.Uvarint(buff)
	if lenlen <= 0 {
		Warnf("%s: error reading initial packet length, disconnecting", conn.RemoteAddr())
		conn.Close()
	}
	Debugf("packet length: %d", pktlen)
	pkt := buff[lenlen:]
	curlen := n - int(lenlen)
	for ; curlen < int(pktlen) ; {
		now, err := conn.Read(pkt[curlen:])
		if now == 0 {
			Warnf("no data received, disconnecting %s", conn.RemoteAddr())
			Free(buff)
			conn.Close()
			return
		}
		if err != nil {
			Warnf("recv error from %s: %s", conn.RemoteAddr(), err.Error())
			conn.Close()
			Free(buff)
			return
		}
		Debugf("recv %d bytes", now)
		curlen += now
	}
	init_packet := decodeHandshake(pkt[:pktlen])
	if init_packet == nil {
		Warnf("invalid packet, disconnecting %s", conn.RemoteAddr())
		Free(buff)
		conn.Close()
		return
	}
	upstream, err := GetUpstream(init_packet.host)
	if err != nil {
		// TODO: Kick with error
		KickClient(conn, err.Error())
		Free(buff)
		conn.Close()
		return
	}
	go startProxy(conn, upstream, orig[:lenlen + curlen])
}


