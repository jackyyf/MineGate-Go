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

type OldClient byte

func (err OldClient) Error() string {
	return fmt.Sprintf("Unexpected leading byte %x", byte(err))
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

type MCLogin struct {
	Name string
}

type Icon string

type MCKick mcchat.ChatMsg

type MCStatusResponse struct {
	// ID is always 0x00
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
		// Sample [0]int `json:"sample"`
	} `json:"players"`
	Description *mcchat.ChatMsg `json:"description"`
	Favicon     Icon            `json:"favicon,omitempty"`
}

type MCSimpleStatusResponse struct {
	// ID is always 0x00
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
		// Sample [0]int `json:"sample"`
	} `json:"players"`
	Description string `json:"description"`
	Favicon     Icon   `json:"favicon,omitempty"`
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
		if first_byte == '\xFE' /* Status */ || first_byte == '\x02' /* Login */ {
			return nil, OldClient(first_byte)
		}
	}
	return
}

func ReadPacket(r SocketReader) (packet *RAWPacket, err error) {
	log.Debug("mcproto.ReadPacket")
	delta := 0
	pktl, err := binary.ReadUvarint(r)
	log.Debugf("packet length: %d", pktl)
	if err != nil {
		log.Error("Read packet error: " + err.Error())
		return nil, err
	}
	payload := make([]byte, pktl)
	for {
		l, err := r.Read(payload[delta:])
		if err != nil {
			log.Error("Read packet error: " + err.Error())
			return nil, err
		}
		if l == 0 {
			err = errors.New("Read reach EOF.")
			log.Error("Read packet error: " + err.Error())

			return nil, err
		}
		delta += l
		if delta == len(payload) {
			break
		}
	}
	id, l := binary.Uvarint(payload)
	if l <= 0 {
		err = errors.New("Invalid packet id.")
		log.Error("Read packet error: " + err.Error())
		return nil, err
	}
	log.Debugf("packet id: %d", id)
	return &RAWPacket{
		ID:      id,
		Payload: payload[l:],
	}, nil
}

func (pkt *RAWPacket) ToBytes() (packet []byte) {
	log.Debug("mcproto.ToBytes")
	pktpayload := make([]byte, len(pkt.Payload)+binary.MaxVarintLen32)
	vlen := binary.PutUvarint(pktpayload, pkt.ID)
	log.Debugf("id length: %d", vlen)
	copy(pktpayload[vlen:], pkt.Payload)
	pktpayload = pktpayload[:vlen+len(pkt.Payload)]
	log.Debugf("packet total length: %d", len(pktpayload))
	buff := make([]byte, len(pktpayload)+binary.MaxVarintLen32)
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

func ReadMCByteString(buff []byte) (bstr []byte, length int, err error) {
	l, delta := binary.Uvarint(buff)
	if delta <= 0 {
		return nil, -1, errors.New("Invalid string length.")
	}
	buff = buff[delta:]
	if len(buff) < int(l) {
		return nil, -1, errors.New("No enough data to read.")
	}
	delta += int(l)
	return buff[:l], delta, nil
}

func WriteMCString(str string) (payload []byte) {
	bs := []byte(str)
	payload = make([]byte, len(bs)+binary.MaxVarintLen32)
	l := binary.PutUvarint(payload, uint64(len(bs)))
	copy(payload[l:], bs)
	return payload[:l+len(bs)]
}

func WriteMCByteString(bstr []byte) (payload []byte) {
	payload = make([]byte, len(bstr)+binary.MaxVarintLen32)
	l := binary.PutUvarint(payload, uint64(len(bstr)))
	copy(payload[l:], bstr)
	return payload[:l+len(bstr)]
}

func (pkt *RAWPacket) IsStatusRequest() (status_req bool) {
	return pkt.ID == 0 && len(pkt.Payload) == 0
}

func (pkt *RAWPacket) IsStatusPing() (status_ping bool) {
	return pkt.ID == 1 && len(pkt.Payload) == 8 /* A long */
}

func (pkt *RAWPacket) ToKick() (kick *MCKick, err error) {
	defer func() {
		// Do not panic please :)
		if r := recover(); r != nil {
			log.Errorf("Recovered from panic: %s", r)
			kick = nil
			err = errors.New("Recovered from panic!")
			return
		}
	}()
	if pkt.ID != 0 {
		return nil, errors.New("Packet ID is not 0.")
	}
	s, l, err := ReadMCByteString(pkt.Payload)
	if err != nil {
		return nil, err
	}
	if len(pkt.Payload) != l {
		return nil, errors.New("Invalid packet: extra field.")
	}
	err = json.Unmarshal(s, kick)
	if err != nil {
		return nil, err
	}
	return kick, nil
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
	json_data, l, err := ReadMCByteString(pkt.Payload)
	if err != nil {
		return nil, err
	}
	pkt.Payload = pkt.Payload[l:]
	if len(pkt.Payload) > 0 {
		return nil, fmt.Errorf("Unexpected extra %d bytes data.", len(pkt.Payload))
	}
	resp = new(MCStatusResponse)
	err = json.Unmarshal(json_data, resp)
	if err != nil {
		if _, ok := err.(*json.UnmarshalTypeError); ok {
			simple_resp := new(MCSimpleStatusResponse)
			err = json.Unmarshal(json_data, simple_resp)
			if err != nil {
				log.Debugf("json_data: %s", string(json_data))
				return nil, err
			}
			resp.Version = simple_resp.Version
			resp.Players = simple_resp.Players
			resp.Favicon = simple_resp.Favicon
			resp.Description = mcchat.NewMsg(simple_resp.Description)
		} else {
			log.Debugf("json_data: %s", string(json_data))
			return nil, err
		}
	}
	return resp, nil
}

func (pkt *RAWPacket) ToLogin() (login *MCLogin, err error) {
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
	name, l, err := ReadMCString(pkt.Payload)
	if err != nil {
		return nil, err
	}
	if len(pkt.Payload) != l {
		return nil, fmt.Errorf("Unexpected extra %d bytes data.", len(pkt.Payload)-l)
	}
	login = new(MCLogin)
	login.Name = name
	return login, nil
}

func (kick *MCKick) ToRawPacket() (pkt *RAWPacket, err error) {
	json_str, err := json.Marshal(kick)
	if err != nil {
		return nil, err
	}
	pkt = new(RAWPacket)
	pkt.ID = 0
	pkt.Payload = WriteMCByteString(json_str)
	return pkt, nil
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
	//addr_bytes := []byte(handshake.ServerAddr)
	addr_bytes := WriteMCString(handshake.ServerAddr)
	payload := make([]byte, binary.MaxVarintLen32 /* protocol */ +len(addr_bytes) /* hostname */ +
		2 /* port */ +binary.MaxVarintLen32 /* nextstate */)
	l := 0
	l += binary.PutUvarint(payload[l:], handshake.Proto)
	l += copy(payload[l:], addr_bytes)
	binary.BigEndian.PutUint16(payload[l:], handshake.ServerPort)
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
	log.Debug("json data: " + string(data))
	if err != nil {
		return nil, err
	}
	return &RAWPacket{
		ID:      0,
		Payload: WriteMCByteString(data),
	}, nil
}

func (login *MCLogin) ToRawPacket() (pkt *RAWPacket, err error) {
	defer func() {
		// Do not panic please :)
		if r := recover(); r != nil {
			log.Errorf("Recovered from panic: %s", r)
			pkt = nil
			err = errors.New("Recovered from panic.")
			return
		}
	}()
	if login == nil {
		return nil, errors.New("Nil login packet.")
	}
	return &RAWPacket{
		ID:      0,
		Payload: WriteMCString(login.Name),
	}, nil
}

/*

	if buff[0] == 0xFE || buff[0] == 0x02 {
		log.Warnf("%s: using old (1.6-) protocol, disconnecting", conn.RemoteAddr())
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
*/
