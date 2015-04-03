package minegate

import (
	"errors"
	"github.com/jackyyf/MineGate-Go/mcchat"
	"github.com/jackyyf/MineGate-Go/mcproto"
	log "github.com/jackyyf/golog"
	"io"
	"net"
	"time"
)

var total_online uint32

func PipeIt(reader *WrapedSocket, writer *WrapedSocket) {
	// TODO: Use configurable buffer size.
	raddr := reader.RemoteAddr().String()
	waddr := writer.RemoteAddr().String()
	log.Infof("%s ==PIPE==> %s", raddr, waddr)
	defer writer.Close()
	buffer := make([]byte, 4096)
	for {
		reader.SetTimeout(15 * time.Second)
		n, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				reader.Warnf("EOF, closing connection.")
			} else {
				reader.Errorf("read error: %s", err.Error())
			}
			reader.Close()
			if n > 0 {
				writer.SetWriteTimeout(15 * time.Second)
				writer.Write(buffer[:n])
			}
			return
		}
		writer.SetWriteTimeout(15 * time.Second)
		n, err = writer.Write(buffer[:n])
		if err != nil {
			writer.Errorf("write error: %s", err.Error())
			return
		}
		// log.Debugf("%s == %d bytes => %s", raddr, n, waddr)
	}
}

func RejectHandler(conn *WrapedSocket, initial_pkt *mcproto.MCHandShake, e *mcchat.ChatMsg) {
	if initial_pkt.NextState == 1 {
		conn.Infof("ping packet")
		pkt, err := mcproto.ReadPacket(conn)
		if err != nil {
			conn.Errorf("Error when reading status request: %s", err.Error())
			conn.Close()
			return
		}
		if !pkt.IsStatusRequest() {
			conn.Errorf("Invalid protocol: no status request.")
			conn.Close()
			return
		}
		conn.Debugf("status: request")
		resp := new(mcproto.MCStatusResponse)
		resp.Description = e
		resp.Version.Name = "minegate"
		resp.Version.Protocol = 0
		resp_pkt, err := resp.ToRawPacket()
		if err != nil {
			conn.Errorf("Unable to make packet: %s", err.Error())
			conn.Close()
			return
		}
		_, err = conn.Write(resp_pkt.ToBytes())
		if err != nil {
			log.Errorf("Unable to write response: %s", err.Error())
			conn.Close()
			return
		}
		pkt, err = mcproto.ReadPacket(conn)
		if err != nil {
			if err != io.EOF {
				log.Errorf("Unable to read packet: %s", err.Error())
			}
			conn.Close()
			return
		}
		if !pkt.IsStatusPing() {
			log.Error("Invalid protocol: no status ping.")
			conn.Close()
			return
		}
		conn.Write(pkt.ToBytes()) // Don't care now.
	} else {
		log.Info("login packet")
		kick_pkt := (*mcproto.MCKick)(e)
		raw_pkt, err := kick_pkt.ToRawPacket()
		if err != nil {
			log.Errorf("Unable to make packet: %s", err.Error())
			conn.Close()
			return
		}
		// Don't care now
		conn.Write(raw_pkt.ToBytes())
	}
	conn.Close()
	return
}

func proxy(conn *WrapedSocket, upstream *Upstream, initial_pkt *mcproto.MCHandShake, ne *PostAcceptEvent) {
	addr, perr := net.ResolveTCPAddr("tcp", upstream.Server)
	var err error
	var upsock *net.TCPConn
	if perr == nil {
		upsock, err = net.DialTCP("tcp", nil, addr)
	}
	if err != nil || perr != nil {
		if err == nil {
			err = perr
		}
		log.Errorf("Unable to connect to upstream %s", upstream.Server)
		RejectHandler(conn, initial_pkt, upstream.ChatMsg)
		return
	}
	upconn := WrapUpstreamSocket(upsock, conn)
	if initial_pkt.NextState == 1 {
		// Handle ping here.
		conn.Debugf("ping proxy")
		pre := new(PingRequestEvent)
		pre.NetworkEvent = ne.NetworkEvent
		pre.Packet = initial_pkt
		pre.Upstream = upstream
		PingRequest(pre)
		if pre.Rejected() {
			if pre.reason == "" {
				conn.Warnf("Ping request was rejected.")
				pre.reason = "Request was rejected by plugin."
			} else {
				conn.Warnf("Ping request was rejected: %s", pre.reason)
			}
			e := mcchat.NewMsg(pre.reason)
			e.SetColor(mcchat.RED)
			e.SetBold(true)
			RejectHandler(conn, initial_pkt, e)
			return
		}
		init_raw, err := initial_pkt.ToRawPacket()
		if err != nil {
			log.Errorf("Unable to encode initial packet: %s", err.Error())
			conn.Close()
			upconn.Close()
			return
		}
		pkt, err := mcproto.ReadPacket(conn)
		if err != nil {
			conn.Errorf("Error when reading status request: %s", err.Error())
			conn.Close()
			upconn.Close()
			return
		}
		if !pkt.IsStatusRequest() {
			conn.Errorf("Invalid protocol: no status request.")
			conn.Close()
			upconn.Close()
			return
		}
		_, err = upconn.Write(init_raw.ToBytes())
		if err == nil {
			_, err = upconn.Write(pkt.ToBytes())
		}
		if err != nil {
			upconn.Errorf("write error: %s", err.Error())
			conn.Close()
			upconn.Close()
			return
		}
		resp_pkt, err := mcproto.ReadPacket(upconn)
		if err != nil {
			upconn.Errorf("invalid packet: %s", err.Error())
			conn.Close()
			upconn.Close()
			return
		}
		resp, err := resp_pkt.ToStatusResponse()
		if err != nil {
			upconn.Errorf("invalid packet: %s", err.Error())
			conn.Close()
			upconn.Close()
			return
		}
		psre := new(PreStatusResponseEvent)
		psre.NetworkEvent = ne.NetworkEvent
		psre.Packet = resp
		psre.Upstream = upstream
		PreStatusResponse(psre)
		resp_pkt, err = resp.ToRawPacket()
		if err != nil {
			conn.Errorf("invalid packet: %s", err.Error())
			conn.Close()
			upconn.Close()
			return
		}
		// We can handle ping request, close upstream
		upconn.Close()
		_, err = conn.Write(resp_pkt.ToBytes())
		if err != nil {
			conn.Errorf("write error: %s", err.Error())
			conn.Close()
			return
		}
		ping_pkt, err := mcproto.ReadPacket(conn)
		if err != nil || !ping_pkt.IsStatusPing() {
			if err == nil {
				err = errors.New("packet is not ping")
			}
			conn.Errorf("invalid packet: %s", err.Error())
			conn.Close()
			return
		}
		_, err = conn.Write(ping_pkt.ToBytes())
		conn.Close()
	} else {
		// Handle login here.
		conn.Debugf("login proxy")
		login_raw, err := mcproto.ReadPacket(conn)
		if err != nil {
			conn.Errorf("Read login packet: %s", err.Error())
			conn.Close()
			return
		}
		login_pkt, err := login_raw.ToLogin()
		if err != nil {
			conn.Errorf("invalid packet: %s", err.Error())
			conn.Close()
			return
		}
		lre := new(LoginRequestEvent)
		lre.NetworkEvent = ne.NetworkEvent
		lre.InitPacket = initial_pkt
		lre.LoginPacket = login_pkt
		lre.Upstream = upstream
		LoginRequest(lre)
		if lre.Rejected() {
			if lre.reason == "" {
				conn.Warnf("Ping request was rejected.")
				lre.reason = "Request was rejected by plugin."
			} else {
				conn.Warnf("Ping request was rejected: %s", lre.reason)
			}
			e := mcchat.NewMsg(lre.reason)
			e.SetColor(mcchat.RED)
			e.SetBold(true)
			RejectHandler(conn, initial_pkt, e)
			return
		}
		init_raw, err := initial_pkt.ToRawPacket()
		if err != nil {
			log.Errorf("Unable to encode initial packet: %s", err.Error())
			conn.Close()
			return
		}
		login_raw, err = login_pkt.ToRawPacket()
		if err != nil {
			log.Errorf("Unable to encode login packet: %s", err.Error())
			conn.Close()
			return
		}
		_, err = upconn.Write(init_raw.ToBytes())
		if err == nil {
			upconn.Write(login_raw.ToBytes())
		}
		if err != nil {
			upconn.Errorf("write error: %s", err.Error())
		}
		spe := new(StartProxyEvent)
		spe.Upstream = upstream
		spe.NetworkEvent = ne.NetworkEvent
		spe.InitPacket = initial_pkt
		spe.LoginPacket = login_pkt
		StartProxy(spe)
		go PipeIt(conn, upconn)
		go PipeIt(upconn, conn)
	}
}

func ServerSocket() {
	addr, err := net.ResolveTCPAddr("tcp", config.Listen_addr)
	if err != nil {
		log.Fatalf("error parse address %s: %s", config.Listen_addr, err.Error())
	}
	s, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatalf("error listening on %s: %s", config.Listen_addr, err.Error())
	}
	log.Infof("Server listened on %s", config.Listen_addr)
	for {
		conn, err := s.AcceptTCP()
		if err != nil {
			log.Warnf("listen_socket: error when accepting: %s", err.Error())
			continue
		}
		go func(conn *WrapedSocket) {
			event := new(PostAcceptEvent)
			event.RemoteAddr = conn.RemoteAddr().(*net.TCPAddr)
			event.connID = conn.Id()
			PostAccept(event)
			if event.Rejected() {
				conn.Warnf("connection rejected.")
				conn.Close()
				return
			}
			log.Infof("connection accepted.")
			ClientSocket(conn, event)
		}(WrapClientSocket(conn))
	}
}

func ClientSocket(conn *WrapedSocket, ne *PostAcceptEvent) {
	init_pkt, err := mcproto.ReadInitialPacket(conn)
	if err != nil {
		if mcproto.IsOldClient(err) {
			// TODO: Implements 1.6- protocol.
			conn.Warnf("1.6- protocol")
			b, _ := err.(mcproto.OldClient)
			if b == 0x02 { // Login
			} else { // Status
			}
			conn.Close()
			return
		} else {
			conn.Errorf("error reading first packet: %s", err.Error())
			conn.Close()
			return
		}
	}
	handshake, err := init_pkt.ToHandShake()
	if err != nil {
		conn.Errorf("Invalid handshake packet: %s", err.Error())
		conn.Close()
		return
	}
	pre := new(PreRoutingEvent)
	pre.NetworkEvent = ne.NetworkEvent
	pre.Packet = handshake
	PreRouting(pre)
	if pre.Rejected() {
		if pre.reason == "" {
			conn.Warnf("Routing request was rejected.")
			pre.reason = "Request was rejected by plugin."
		} else {
			conn.Warnf("Routing request was rejected: %s", pre.reason)
		}
		e := mcchat.NewMsg(pre.reason)
		e.SetColor(mcchat.RED)
		e.SetBold(true)
		RejectHandler(conn, handshake, e)
	} else {
		upstream, e := GetUpstream(handshake.ServerAddr)
		if e != nil {
			RejectHandler(conn, handshake, e)
			return
		}
		proxy(conn, upstream, handshake, ne)
	}

}
