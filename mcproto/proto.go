package mcproto

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	mcchat "github.com/jackyyf/MineGate-Go/mcchat"
	log "github.com/jackyyf/golog"
	"image/png"
	"io"
	"io/ioutil"
)

type RAWPacket struct {
	ID      uint64
	Payload []byte
}

const (
	HandShakeID uint64 = 0
)

var transparent_png = []byte(`"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNgYAAAAAMAASsJTYQAAAAASUVORK5CYII="`)
var prefix = `"data:image/png;base64,`
var suffix = `"`

var prefix_len = len(prefix)
var suffix_len = len(suffix)

type MCPacket interface {
	ToRawPacket() (*RAWPacket, error)
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
	Favicon     Icon           `json:",omitempty"`
}

func (icon Icon) MarshalJSON() (payload []byte, err error) {
	// @TODO: Read icon file each time is insufficient, add an IconManager.
	content, err := ioutil.ReadFile(string(icon))
	if err != nil {
		// Never fail, return a empty png, and issue an error.
		if icon != "" {
			log.Error("Unable to open icon file " + string(icon))
		}
		return transparent_png, nil
	}
	// partial check png, read only config, instead of whole image.
	_, err = png.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		log.Error("Failed to decode png.")
		return transparent_png, nil
	}
	payload = make([]byte, prefix_len+base64.StdEncoding.EncodedLen(len(content))+suffix_len)
	copy(payload, prefix)
	base64.StdEncoding.Encode(payload[prefix_len:], content)
	copy(payload[prefix_len+base64.StdEncoding.EncodedLen(len(content)):], suffix)
	return payload, nil
}

func ReadPacket(r io.Reader) (packet *RAWPacket, err error) {
	var payload []byte
	ldelta := 0
	delta := 0
	buff := make([]byte, binary.MaxVarintLen32)
	for {
		l, err := r.Read(buff[ldelta:])
		ldelta += l
		if err != nil {
			return nil, err
		}
		if l == 0 {
			return nil, errors.New("Read reach EOF.")
		}
		pktl, d := binary.Uvarint(buff)
		if d > 0 {
			payload = make([]byte, pktl)
			if len(buff[d:]) > 0 {
				copy(payload, buff[d:])
				delta = len(buff[d:])
			}
			break
		}
		if len(buff) == 0 {
			return nil, errors.New("Invalid packet length.")
		}
	}
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
	handshake.ServerPort = (uint16(payload[0]) << 8) | uint16(payload[1])
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
	payload[l] = byte(handshake.ServerPort >> 8)
	payload[l+1] = byte(handshake.ServerPort & 255)
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
