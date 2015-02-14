package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/jackyyf/MineGate-Go/mcchat"
	log "github.com/jackyyf/golog"
	"net"
)

type Handshake struct {
	version int
	host    string
	port    uint16
	isPing  bool
}

type MinecraftVersion struct {
	Name     string `json:"name"`
	Protocol int    `json:"protocol"`
}

type MinecraftPlayerStatus struct {
	Max    int     `json:"max"`
	Online int     `json:"online"`
	Sample [0]byte `json:"sample"`
}

type StatusResponse struct {
	Version     MinecraftVersion      `json:"version"`
	Players     MinecraftPlayerStatus `json:"players"`
	Description *mcchat.ChatMsg       `json:"description"`
	Favicon     string                `json:"favicon"`
}

func KickClient(conn *net.TCPConn, msg string) {
	buffer := bytes.NewBuffer(make([]byte, 0, 16384))
	buffer.Write([]byte{'\x00'})
	buffer.Write(mcchat.NewMsg(msg).AsChatString())
	length := uint64(buffer.Len())
	lendata := make([]byte, binary.MaxVarintLen32)
	conn.Write(lendata[:binary.PutUvarint(lendata, length)])
	conn.Write(buffer.Bytes())
}

func KickClientRaw(conn *net.TCPConn, msg []byte) {
	lendata := make([]byte, binary.MaxVarintLen32)
	conn.Write(lendata[:binary.PutUvarint(lendata, uint64(len(msg)+1))])
	conn.Write([]byte{'\x00'})
	conn.Write(msg)
}

func ResponsePing(conn *net.TCPConn, msg *mcchat.ChatMsg) {
	response, _ := json.Marshal(StatusResponse{
		Version: MinecraftVersion{
			Name:     "minegate",
			Protocol: 23333, // Use a specialized version to indicate server is not joinable.
		},
		Players: MinecraftPlayerStatus{
			Max:    1,
			Online: 1,
		},
		Description: msg,
		Favicon:     "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAAAAAA6fptVAAAACklEQVQYV2P4DwABAQEAWk1v8QAAAABJRU5ErkJggg==",
	})
	rlen := len(response)
	lbuf := make([]byte, binary.MaxVarintLen32)
	llen := binary.PutUvarint(lbuf, uint64(rlen))
	log.Debug("ping.response = " + string(response))
	lendata := make([]byte, binary.MaxVarintLen32)
	conn.Write(lendata[:binary.PutUvarint(lendata, uint64(rlen+1+llen))])
	conn.Write([]byte{'\x00'})
	conn.Write(lbuf[:llen])
	conn.Write(response)
}

func decodeHandshake(pkt []byte) (handshake *Handshake) {
	if pkt[0] != 0 {
		log.Warn("Invalid packet: handshake pkt id should be 0.")
	}
	pkt = pkt[1:]
	proto, vlen := binary.Uvarint(pkt)
	if vlen <= 0 {
		log.Warn("Invalid packet: invalid protocol version")
		return nil
	}
	pkt = pkt[vlen:]
	hlen, vlen := binary.Uvarint(pkt)
	if vlen <= 0 {
		log.Warn("Invalid packet: invalid hostname length")
		return nil
	}
	pkt = pkt[vlen:]
	if int(hlen) >= len(pkt) {
		log.Warnf("String too long: %d > %d", hlen, len(pkt))
		return nil
	}
	host := pkt[:hlen]
	pkt = pkt[hlen:]
	if len(pkt) < 3 {
		log.Warn("No enough data to decode.")
		return nil
	}
	var port uint16 = uint16(pkt[0])<<8 | uint16(pkt[1])
	pkt = pkt[2:]
	state, vlen := binary.Uvarint(pkt)
	if vlen <= 0 {
		log.Warn("Invalid packet: invalid next state")
		return nil
	}
	if state < 1 || state > 2 {
		log.Warnf("Invalid next state: %d", state)
		return nil
	}
	pkt = pkt[vlen:]
	if state == 1 && len(pkt) > 1 {
		if pkt[0] == 1 && pkt[1] == 0 {
			pkt = pkt[2:]
		}
	}
	return &Handshake{
		version: int(proto),
		host:    string(host),
		port:    port,
		isPing:  state == 1,
	}
}
