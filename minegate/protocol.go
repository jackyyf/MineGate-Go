package main

import (
	"encoding/binary"
	"encoding/json"
	"bytes"
	"net"
)

func makeChatMsg(msg string) (res []byte) {
	// Should not throw an error.
	res, _ = json.Marshal(map[string]string {
		"text": msg,
	})
	msglen := len(res)
	buff := make([]byte, msglen + 5)
	vintlen := binary.PutUvarint(buff, uint64(msglen))
	copy(buff[vintlen:], res)
	return buff[:vintlen + msglen]
}

func KickClient(conn net.Conn, msg string) {
	buffer := bytes.NewBuffer(make([]byte, 0, 16384))
	buffer.Write([]byte{'\x00'})
	buffer.Write(makeChatMsg(msg))
	length := uint64(buffer.Len())
	lendata := make([]byte, 10)
	conn.Write(lendata[:binary.PutUvarint(lendata, length)])
	conn.Write(buffer.Bytes())
}

func decodeHandshake(pkt []byte) (handshake *Handshake) {
	if pkt[0] != 0 {
		Warn("Invalid packet: handshake pkt id should be 0.")
	}
	pkt = pkt[1:]
	proto, vlen := binary.Uvarint(pkt)
	if vlen <= 0 {
		Warn("Invalid packet: invalid protocol version")
		return nil
	}
	pkt = pkt[vlen:]
	hlen, vlen := binary.Uvarint(pkt)
	if vlen <= 0 {
		Warn("Invalid packet: invalid hostname length")
		return nil
	}
	pkt = pkt[vlen:]
	if int(hlen) >= len(pkt) {
		Warnf("String too long: %d > %d", hlen, len(pkt))
		return nil
	}
	host := pkt[:hlen]
	pkt = pkt[hlen:]
	if len(pkt) < 3 {
		Warn("No enough data to decode.")
		return nil
	}
	var port uint16 = uint16(pkt[0]) << 8 | uint16(pkt[1])
	pkt = pkt[2:]
	state, vlen := binary.Uvarint(pkt)
	if vlen <= 0 {
		Warn("Invalid packet: invalid next state")
		return nil
	}
	if state < 1 || state > 2 {
		Warnf("Invalid next state: %d", state)
		return nil
	}
	return &Handshake {
		version: int(proto),
		host: string(host),
		port: port,
		isPing: state == 1,
	}
}
