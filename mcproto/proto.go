package mcproto

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	mcchat "github.com/jackyyf/MineGate-Go/mcchat"
	log "github.com/jackyyf/golog"
	"io"
	"strings"
)

type RAWPacket struct {
	ID      uint64
	Payload []byte
}

const (
	HandShakeID uint64 = 0
)

var transparent_png = []byte(`data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNgYAAAAAMAASsJTYQAAAAASUVORK5CYII=`)
var prefix = `data:image/png;base64,`

var prefix_len = len(prefix)

type OldClient string

func (err OldClient) Error() string {
	return string(err)
}

type MCPacket interface {
	ToRawPacket() (*RAWPacket, error)
}

type SocketReader interface {
	io.Reader
	io.ByteScanner
}

type MCHandShake struct {
	// ID is always 0x00
	Proto      uint64
	ServerAddr string
	ServerPort uint16
	NextState  uint64
}

type Icon string

type MCStatusResponse struct {
	// ID is always 0x00
	Version struct {
		Name     string `json:name`
		Protocol int    `json:protocol`
	} `json:version`
	Players struct {
		Max    int `json:max`
		Online int `json:online`
		// Sample [0]int `json:sample`
	} `json:players`
	Description mcchat.ChatMsg `json:description`
	Favicon     Icon           `json:"favicon,omitempty"`
}

func (icon Icon) ToBinaryImage() (img []byte, err error) {
	if !strings.HasPrefix(string(icon), prefix) {
		return nil, errors.New("Invalid base64 icon data.")
	}
	data := []byte(icon[len(prefix):])
	img = make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	base64.StdEncoding.Decode(img, data)
	return img, nil
}

func IsOldClient(err error) (old bool) {
	_, old = err.(OldClient)
	return
}

func ReadInitialPacket(r SocketReader) (packet *RAWPacket, err error) {
	first_byte, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	err = r.UnreadByte()
	if err != nil {
		return nil, err
	}
	packet, err = ReadPacket(r)
	if err != nil {
		if first_byte == '\xFE' {
			return nil, OldClient("First byte is 0xFE.")
		}
	}
	return
}

func ReadPacket(r SocketReader) (packet *RAWPacket, err error) {
	delta := 0
	pktl, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, err
	}
	payload := make([]byte, pktl)
	for {
		l, err := r.Read(payload[delta:])
		if err != nil {
			return nil, err
		}
		if l == 0 {
			return nil, errors.New("Read reach EOF.")
		}
		delta += l
		if delta == len(payload) {
			break
		}
	}
	id, l := binary.Uvarint(payload)
	if l <= 0 {
		return nil, errors.New("Invalid packet id.")
	}
	return &RAWPacket{
		ID:      id,
		Payload: payload[l:],
	}, nil
}

func (pkt *RAWPacket) ToBytes() (packet []byte) {
	pktpayload := make([]byte, len(pkt.Payload)+binary.MaxVarintLen32)
	vlen := binary.PutUvarint(pktpayload, pkt.ID)
	copy(pktpayload[vlen:], pkt.Payload)
	pktpayload = pktpayload[:vlen+len(pkt.Payload)]
	buff := make([]byte, len(pktpayload)+binary.MaxVarintLen32*2)
	vlen = binary.PutUvarint(buff, uint64(len(pktpayload)))
	copy(buff[vlen:], pktpayload)
	return buff[:vlen+len(pktpayload)]
}

func ReadMCString(buff []byte) (str string, length int, err error) {
	l, delta := binary.Uvarint(buff)
	if delta <= 0 {
		return "", -1, errors.New("Invalid string length.")
	}
	buff = buff[delta:]
	if len(buff) < int(l) {
		return "", -1, errors.New("No enough data to read.")
	}
	delta += int(l)
	return string(buff[:l]), delta, nil
}

func (pkt *RAWPacket) ToHandShake() (handshake *MCHandShake, err error) {
	defer func() {
		// Do not panic please :)
		if r := recover(); r != nil {
			log.Errorf("Recovered from panic: %s", r)
			handshake = nil
			err = errors.New("Recovered from panic!")
			return
		}
	}()
	if pkt.ID != 0 {
		return nil, fmt.Errorf("Invalid packet id: %d, should be 0.", pkt.ID)
	}
	handshake = new(MCHandShake)
	payload := pkt.Payload
	var l int
	handshake.Proto, l = binary.Uvarint(payload)
	if l <= 0 {
		return nil, errors.New("Invalid protocol.")
	}
	payload = payload[l:]
	str, l, err := ReadMCString(payload)
	if err != nil {
		return nil, errors.New("Invalid hostname: " + err.Error())
	}
	payload = payload[l:]
	handshake.ServerAddr = str
	if len(payload) < 2 {
		return nil, errors.New("Invalid port: no enough data.")
	}
	// handshake.ServerPort = (uint16(payload[0]) << 8) | uint16(payload[1])
	handshake.ServerPort = binary.BigEndian.Uint16(payload)
	payload = payload[2:]
	nextState, l := binary.Uvarint(payload)
	if l <= 0 {
		return nil, errors.New("Invalid nextstate.")
	}
	handshake.NextState = nextState
	payload = payload[l:]
	if len(payload) != 0 {
		return nil, errors.New("Invalid packet: unexpected extra data.")
	}
	return handshake, nil
}

func (pkt *RAWPacket) ToStatusResponse() (resp *MCStatusResponse, err error) {
	defer func() {
		// Do not panic please :)
		if r := recover(); r != nil {
			log.Errorf("Recovered from panic: %s", r)
			pkt = nil
			err = errors.New("Recovered from panic.")
			return
		}
	}()
	if pkt.ID != 0 {
		return nil, fmt.Errorf("Unexpected packet id: %d", pkt.ID)
	}
	json_data, l, err := ReadMCString(pkt.Payload)
	pkt.Payload = pkt.Payload[l:]
	if len(pkt.Payload) > 0 {
		return nil, fmt.Errorf("Unexpected extra %d bytes data.", len(pkt.Payload))
	}
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(json_data), resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (handshake *MCHandShake) ToRawPacket() (pkt *RAWPacket, err error) {
	defer func() {
		// Do not panic please :)
		if r := recover(); r != nil {
			log.Errorf("Recovered from panic: %s", r)
			pkt = nil
			err = errors.New("recover from panic")
			return
		}
	}()
	if handshake == nil {
		return nil, errors.New("Nil handshake")
	}
	pkt = new(RAWPacket)
	pkt.ID = HandShakeID
	addr_bytes := []byte(handshake.ServerAddr)
	payload := make([]byte, binary.MaxVarintLen32 /* protocol */ +binary.MaxVarintLen32+len(addr_bytes) /* hostname */ +
		2 /* port */ +binary.MaxVarintLen32 /* nextstate */)
	l := 0
	l += binary.PutUvarint(payload[l:], handshake.Proto)
	l += binary.PutUvarint(payload[l:], uint64(len(addr_bytes)))
	copy(payload[l:], addr_bytes)
	l += len(addr_bytes)
	l += 2
	l += binary.PutUvarint(payload[l:], handshake.NextState)
	pkt.Payload = payload[:l]
	return pkt, nil
}

func (resp *MCStatusResponse) ToRawPacket() (pkt *RAWPacket, err error) {
	defer func() {
		// Do not panic please :)
		if r := recover(); r != nil {
			log.Errorf("Recovered from panic: %s", r)
			pkt = nil
			err = errors.New("Recovered from panic.")
			return
		}
	}()
	if resp == nil {
		return nil, errors.New("Nil response packet.")
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}
	payload := make([]byte, binary.MaxVarintLen32+len(data))
	vl := binary.PutUvarint(payload, uint64(len(data)))
	copy(payload[vl:], data)
	return &RAWPacket{
		ID:      0,
		Payload: payload[:vl+len(data)],
	}, nil
}
