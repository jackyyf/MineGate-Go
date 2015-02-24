package minegate

import (
	"bufio"
	"errors"
	"github.com/jackyyf/MineGate-Go/mcproto"
	log "github.com/jackyyf/golog"
	"io"
	"net"
	"reflect"
	"time"
)

var total_online uint32

func SetWriteTimeout(conn *net.TCPConn, t time.Duration) {
	// TODO: Use configurable write deadline
	conn.SetWriteDeadline(time.Now().Add(t))
}

func PipeIt(reader *bufio.ReadWriter, writer *bufio.ReadWriter, rsock, wsock func() *net.TCPConn) {
	// TODO: Use configurable buffer size.
	raddr := rsock().RemoteAddr().String()
	waddr := wsock().RemoteAddr().String()
	log.Infof("%s ==PIPE==> %s", raddr, waddr)
	defer wsock().Close()
	buffer := make([]byte, 4096)
	for {
		n, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				log.Warnf("%s read reach EOF, closing connection.", raddr)
			} else {
				log.Errorf("%s read error: %s", raddr, err.Error())
			}
			log.Infof("Closed connection %s", raddr)
			rsock().Close()
			if n > 0 {
				SetWriteTimeout(wsock(), 15*time.Second)
				writer.Write(buffer[:n])
				writer.Flush()
			}
			return
		}
		// log.Debugf("%s => %d bytes.", rsock().RemoteAddr(), n)
		n, err = writer.Write(buffer[:n])
		if err == nil {
			err = writer.Flush()
		}
		if err != nil {
			log.Errorf("%s write error: %s", waddr, err.Error())
			return
		}
		log.Debugf("%s == %d bytes => %s", raddr, n, waddr)
	}
}

func startProxy(conn *bufio.ReadWriter, sock func() *net.TCPConn, upstream *Upstream, initial_pkt *mcproto.MCHandShake, ne *PostAcceptEvent) {
	addr, perr := net.ResolveTCPAddr("tcp", upstream.Server)
	var err error
	var upconn *net.TCPConn
	if perr == nil {
		upconn, err = net.DialTCP("tcp", nil, addr)
	}
	if err != nil || perr != nil {
		if err == nil {
			err = perr
		}
		log.Errorf("Unable to connect to upstream %s", upstream.Server)
		// KickClient(conn, "502 Bad Gateway.")
		if initial_pkt.NextState == 1 {
			log.Info("ping packet")
			pkt, err := mcproto.ReadPacket(conn)
			if err != nil {
				log.Errorf("Error when reading status request: %s", err.Error())
				sock().Close()
				return
			}
			if !pkt.IsStatusRequest() {
				log.Error("Invalid protocol: no status request.")
				sock().Close()
				return
			}
			log.Debug("status: request")
			resp := new(mcproto.MCStatusResponse)
			resp.Description = upstream.ChatMsg
			resp.Version.Name = "minegate"
			resp.Version.Protocol = 0
			resp_pkt, err := resp.ToRawPacket()
			if err != nil {
				log.Errorf("Unable to make packet: %s", err.Error())
				sock().Close()
				return
			}
			_, err = conn.Write(resp_pkt.ToBytes())
			if err == nil {
				err = conn.Flush()
			}
			if err != nil {
				log.Errorf("Unable to write response: %s", err.Error())
				sock().Close()
				return
			}
			pkt, err = mcproto.ReadPacket(conn)
			if err != nil {
				if err != io.EOF {
					log.Errorf("Unable to read packet: %s", err.Error())
				}
				sock().Close()
				return
			}
			if !pkt.IsStatusPing() {
				log.Error("Invalid protocol: no status ping.")
				sock().Close()
				return
			}
			conn.Write(pkt.ToBytes()) // Don't care now.
			conn.Flush()

		} else {
			log.Info("login packet")
			kick_pkt := (*mcproto.MCKick)(upstream.ChatMsg)
			raw_pkt, err := kick_pkt.ToRawPacket()
			if err != nil {
				log.Errorf("Unable to make packet: %s", err.Error())
				sock().Close()
				return
			}
			// Don't care now
			conn.Write(raw_pkt.ToBytes())
			conn.Flush()
		}
		sock().Close()
		return
	}
	upconn.SetNoDelay(false)
	upsock := func() *net.TCPConn {
		return upconn
	}
	ubufrw := bufio.NewReadWriter(bufio.NewReader(upconn), bufio.NewWriter(upconn))
	if initial_pkt.NextState == 1 {
		// Handle ping here.
		log.Debug("ping proxy")
		pre := new(PingRequestEvent)
		pre.NetworkEvent = ne.NetworkEvent
		pre.Packet = initial_pkt
		PingRequest(pre)
		init_raw, err := initial_pkt.ToRawPacket()
		if err != nil {
			log.Errorf("Unable to encode initial packet: %s", err.Error())
			sock().Close()
			upsock().Close()
			return
		}
		pkt, err := mcproto.ReadPacket(conn)
		if err != nil {
			log.Errorf("Error when reading status request: %s", err.Error())
			sock().Close()
			upsock().Close()
			return
		}
		if !pkt.IsStatusRequest() {
			log.Error("Invalid protocol: no status request.")
			sock().Close()
			upsock().Close()
			return
		}
		_, err = ubufrw.Write(init_raw.ToBytes())
		if err == nil {
			_, err = ubufrw.Write(pkt.ToBytes())
		}
		if err == nil {
			err = ubufrw.Flush()
		}
		if err != nil {
			log.Errorf("write error: %s", err.Error())
			sock().Close()
			upsock().Close()
			return
		}
		resp_pkt, err := mcproto.ReadPacket(ubufrw)
		if err != nil {
			log.Errorf("invalid packet: %s", err.Error())
			sock().Close()
			upsock().Close()
			return
		}
		resp, err := resp_pkt.ToStatusResponse()
		if err != nil {
			log.Errorf("invalid packet: %s", err.Error())
			log.Debugf("err type: %+v", reflect.TypeOf(err))
			sock().Close()
			upsock().Close()
			return
		}
		psre := new(PreStatusResponseEvent)
		psre.NetworkEvent = ne.NetworkEvent
		psre.Packet = resp
		PreStatusResponse(psre)
		resp_pkt, err = resp.ToRawPacket()
		if err != nil {
			log.Errorf("invalid packet: %s", err.Error())
			sock().Close()
			upsock().Close()
			return
		}
		// We can handle ping request, close upstream
		upsock().Close()
		_, err = conn.Write(resp_pkt.ToBytes())
		if err == nil {
			err = conn.Flush()
		}
		if err != nil {
			log.Errorf("write error: %s", err.Error())
			sock().Close()
			upsock().Close()
			return
		}
		ping_pkt, err := mcproto.ReadPacket(conn)
		if err != nil || !ping_pkt.IsStatusPing() {
			if err == nil {
				err = errors.New("packet is not ping")
			}
			log.Errorf("invalid packet: %s", err.Error())
			sock().Close()
			upsock().Close()
			return
		}
		_, err = conn.Write(ping_pkt.ToBytes())
		if err == nil {
			err = conn.Flush()
		}
		sock().Close()
		// go PipeIt(conn, ubufrw, sock, upsock)
		// go PipeIt(ubufrw, conn, upsock, sock)
	} else {
		// Handle login here.
		log.Debug("login proxy")
		login_raw, err := mcproto.ReadPacket(conn)
		if err != nil {
			log.Errorf("Read login packet: %s", err.Error())
		}
		login_pkt, err := login_raw.ToLogin()
		lre := new(LoginRequestEvent)
		lre.NetworkEvent = ne.NetworkEvent
		lre.InitPacket = initial_pkt
		lre.LoginPacket = login_pkt
		LoginRequest(lre)
		init_raw, err := initial_pkt.ToRawPacket()
		if err != nil {
			log.Errorf("Unable to encode initial packet: %s", err.Error())
			return
		}
		login_raw, err = login_pkt.ToRawPacket()
		if err != nil {
			log.Errorf("Unable to encode login packet: %s", err.Error())
			return
		}
		ubufrw.Write(init_raw.ToBytes())
		ubufrw.Write(login_raw.ToBytes())
		ubufrw.Flush()
		go PipeIt(conn, ubufrw, sock, upsock)
		go PipeIt(ubufrw, conn, upsock, sock)
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
	total_online = 0
	var connID uintptr = 0
	for {
		conn, err := s.AcceptTCP()
		if err != nil {
			log.Warnf("listen_socket: error when accepting: %s", err.Error())
			continue
		}
		go func(conn *net.TCPConn) {
			event := new(PostAcceptEvent)
			event.RemoteAddr = conn.RemoteAddr()
			event.connID = connID
			connID++
			PostAccept(event)
			if event.Rejected() {
				log.Warnf("Connection ID %d, from %s was rejected.", event.connID, event.RemoteAddr)
				conn.Close()
				return
			}
			conn.SetNoDelay(false)
			log.Infof("listen_socket: new connection id %d from %s", event.connID, conn.RemoteAddr())
			// TODO: Use configurable buffer size.
			ClientSocket(bufio.NewReadWriter(bufio.NewReaderSize(conn, 4096), bufio.NewWriterSize(conn, 4096)), event, func() *net.TCPConn {
				return conn
			})
		}(conn)
	}
}

func ClientSocket(conn *bufio.ReadWriter, ne *PostAcceptEvent, sock func() *net.TCPConn) {
	init_pkt, err := mcproto.ReadInitialPacket(conn)
	if err != nil {
		if mcproto.IsOldClient(err) {
			// TODO: Implements 1.6- protocol.
			b, _ := err.(mcproto.OldClient)
			if b == 0x02 { // Login
			} else { // Status
			}
			sock().Close()
			return
		} else {
			log.Errorf("error when reading first packet: %s", err.Error())
			sock().Close()
			return
		}
	}
	handshake, err := init_pkt.ToHandShake()
	if err != nil {
		log.Errorf("Invalid handshake packet: %s", err.Error())
		sock().Close()
		return
	}
	pre := new(PreRoutingEvent)
	pre.NetworkEvent = ne.NetworkEvent
	pre.Packet = handshake
	PreRouting(pre)
	upstream, e := GetUpstream(handshake.ServerAddr)
	if e != nil {
		// TODO: Kick with error
		if handshake.NextState == 1 {
			log.Info("ping packet")
			pkt, err := mcproto.ReadPacket(conn)
			if err != nil {
				log.Errorf("Error when reading status request: %s", err.Error())
				sock().Close()
				return
			}
			if !pkt.IsStatusRequest() {
				log.Error("Invalid protocol: no status request.")
				sock().Close()
				return
			}
			log.Debug("status: request")
			resp := new(mcproto.MCStatusResponse)
			resp.Description = e
			resp.Version.Name = "minegate"
			resp.Version.Protocol = 0
			resp_pkt, err := resp.ToRawPacket()
			if err != nil {
				log.Errorf("Unable to make packet: %s", err.Error())
				sock().Close()
				return
			}
			_, err = conn.Write(resp_pkt.ToBytes())
			if err == nil {
				err = conn.Flush()
			}
			if err != nil {
				log.Errorf("Unable to write response: %s", err.Error())
			}
			pkt, err = mcproto.ReadPacket(conn)
			if err != nil {
				if err != io.EOF {
					log.Errorf("Unable to read packet: %s", err.Error())
				}
				sock().Close()
				return
			}
			if !pkt.IsStatusPing() {
				log.Error("Invalid protocol: no status ping.")
				sock().Close()
				return
			}
			conn.Write(pkt.ToBytes()) // Don't care now.
			conn.Flush()
		} else {
			log.Info("login packet")
			kick_pkt := (*mcproto.MCKick)(e)
			raw_pkt, err := kick_pkt.ToRawPacket()
			if err != nil {
				log.Errorf("Unable to make packet: %s", err.Error())
				sock().Close()
				return
			}
			// Don't care now
			conn.Write(raw_pkt.ToBytes())
			conn.Flush()
		}
		sock().Close()
		return
	}
	startProxy(conn, sock, upstream, handshake, ne)
}
