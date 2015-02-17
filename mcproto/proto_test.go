package mcproto

import (
	"bytes"
	"testing"
)

func TestOldClient(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Recovered from panic: %s", r)
			return
		}
	}()
	prepared_pkt := []byte{0xFE, 0x01}
	rawpkt, err := ReadInitialPacket(bytes.NewReader(prepared_pkt))
	if IsOldClient(err) {
		t.Log("Ok, Old client detected.")
	} else if err != nil {
		t.Fatal("Unknown error: " + err.Error())
	} else {
		t.Error("Unexpected successful parse!")
		t.Fatalf("Parsed packet: %+v", rawpkt)
	}
}

func TestParsePing(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Recovered from panic: %s", r)
			return
		}
	}()
	prepared_pkt := []byte{
		0x13, 0x00, 0x2f, 0x0d, 0x73, 0x65, 0x72, 0x76,
		0x65, 0x72, 0x31, 0x2e, 0x6c, 0x6f, 0x63, 0x61,
		0x6c, 0x63, 0xdd, 0x01,
	}
	rawpkt, err := ReadInitialPacket(bytes.NewReader(prepared_pkt))
	if err != nil {
		t.Fatal("Unable to read packet: " + err.Error())
	}
	if rawpkt.ID != 0 {
		t.Fatalf("Packet id should be 0, %d found", rawpkt.ID)
		return
	}
	t.Log("Ok, valid mc packet, with packet id 0")
	handshake, err := rawpkt.ToHandShake()
	if err != nil {
		t.Fatal("Unable to convert to handshake packet: " + err.Error())
	}
	if handshake.Proto != 47 {
		t.Errorf("Protocol mismatch, expect 47, found %d", handshake.Proto)
	} else {
		t.Log("Ok, protocol is 47")
	}
	if handshake.ServerAddr != "server1.local" {
		t.Errorf("Server name mismatch, expect server1.local, found %s", handshake.ServerAddr)
	} else if !t.Failed() {
		t.Log("Ok, server name is server1.local")
	}
	if handshake.ServerPort != 25565 {
		t.Errorf("Server port mismatch, expect 25565, found %s", handshake.ServerPort)
	} else if !t.Failed() {
		t.Log("Ok, server port is 25565")
	}
	if handshake.NextState != 1 {
		t.Errorf("Next state mismatch, expect 1, found %d", handshake.NextState)
	} else if !t.Failed() {
		t.Log("Ok, next state is 1")
	}
	if !t.Failed() {
		t.Log("Ok, packet parsed correctly.")
	}
}

func TestParseLogin(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Recovered from panic: %s", r)
			return
		}
	}()
	prepared_pkt := []byte{
		0x16, 0x00, 0x2f, 0x10, 0x73, 0x65, 0x72, 0x76,
		0x65, 0x72, 0x32, 0x2e, 0x6c, 0x6f, 0x63, 0x61,
		0x6c, 0xe4, 0xb8, 0xad, 0x63, 0xdd, 0x02,
	}
	rawpkt, err := ReadInitialPacket(bytes.NewReader(prepared_pkt))
	if err != nil {
		t.Fatal("Unable to read packet: " + err.Error())
	}
	if rawpkt.ID != 0 {
		t.Fatalf("Packet id should be 0, %d found", rawpkt.ID)
		return
	}
	t.Log("Ok, valid mc packet, with packet id 0")
	handshake, err := rawpkt.ToHandShake()
	if err != nil {
		t.Fatal("Unable to convert to handshake packet: " + err.Error())
	}
	if handshake.Proto != 47 {
		t.Errorf("Protocol mismatch, expect 47, found %d", handshake.Proto)
	} else {
		t.Log("Ok, protocol is 47")
	}
	if handshake.ServerAddr != "server2.local中" {
		t.Errorf("Server name mismatch, expect server2.local中, found %s", handshake.ServerAddr)
	} else if !t.Failed() {
		t.Log("Ok, server name is server2.local中")
	}
	if handshake.ServerPort != 25565 {
		t.Errorf("Server port mismatch, expect 25565, found %s", handshake.ServerPort)
	} else if !t.Failed() {
		t.Log("Ok, server port is 25565")
	}
	if handshake.NextState != 2 {
		t.Errorf("Next state mismatch, expect 2, found %d", handshake.NextState)
	} else if !t.Failed() {
		t.Log("Ok, next state is 2")
	}
	if !t.Failed() {
		t.Log("Ok, packet parsed correctly.")
	}
}
