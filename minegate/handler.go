package minegate

import (
	"bufio"
	"errors"
	"github.com/jackyyf/MineGate-Go/mcproto"
	log "github.com/jackyyf/golog"
	"io"
	"net"
	"time"
)

var total_online uint32

func SetWriteTimeout(conn *net.TCPConn, t time.Duration) {
	// TODO: Use configurable write deadline
	conn.SetWriteDeadline(time.Now().Add(t))
}

func PipeIt(reader *bufio.ReadWriter, writer *bufio.ReadWriter, rsock, wsock func() *net.TCPConn) {
	// TODO: Use configurable buffer size.
	log.Infof("%s ==PIPE==> %s", rsock().RemoteAddr(), wsock().RemoteAddr())
	defer wsock().Close()
	buffer := make([]byte, 4096)
	for {
		n, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				log.Warnf("%s read reach EOF, closing connection.", rsock().RemoteAddr())
			} else {
				log.Errorf("%s read error: %s", rsock().RemoteAddr(), err.Error())
			}
			log.Infof("Closed connection %s", rsock().RemoteAddr())
			rsock().Close()
			if n > 0 {
				SetWriteTimeout(wsock(), 15*time.Second)
				writer.Write(buffer[:n])
				writer.Flush()
			}
			return
		}
		log.Debugf("%s => %d bytes.", rsock().RemoteAddr(), n)
		n, err = writer.Write(buffer[:n])
		if err == nil {
			err = writer.Flush()
		}
		if err != nil {
			log.Errorf("%s write error: %s", wsock().RemoteAddr(), err.Error())
			return
		}
		log.Debugf("%d bytes => %s", n, rsock().RemoteAddr())
	}
}

func startProxy(conn *bufio.ReadWriter, sock func() *net.TCPConn, upstream *Upstream, initial_pkt *mcproto.MCHandShake) {
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
				log.Error("Error when reading status request: " + err.Error())
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
				log.Error("Unable to make packet: " + err.Error())
				sock().Close()
				return
			}
			_, err = conn.Write(resp_pkt.ToBytes())
			if err == nil {
				err = conn.Flush()
			}
			if err != nil {
				log.Error("Unable to write response: " + err.Error())
				sock().Close()
				return
			}
			pkt, err = mcproto.ReadPacket(conn)
			if err != nil {
				if err != io.EOF {
					log.Error("Unable to read packet: " + err.Error())
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
				log.Error("Unable to make packet: " + err.Error())
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
		init_raw, err := initial_pkt.ToRawPacket()
		if err != nil {
			log.Error("Unable to encode initial packet: " + err.Error())
			sock().Close()
			upsock().Close()
			return
		}
		pkt, err := mcproto.ReadPacket(conn)
		if err != nil {
			log.Error("Error when reading status request: " + err.Error())
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
			log.Error("write error: " + err.Error())
			sock().Close()
			upsock().Close()
			return
		}
		resp_pkt, err := mcproto.ReadPacket(ubufrw)
		if err != nil {
			log.Error("invalid packet: " + err.Error())
			sock().Close()
			upsock().Close()
			return
		}
		resp, err := resp_pkt.ToStatusResponse()
		if err != nil {
			log.Error("invalid packet: " + err.Error())
			log.Debug(string(resp_pkt.Payload))
			sock().Close()
			upsock().Close()
			return
		}
		resp_pkt, err = resp.ToRawPacket()
		if err != nil {
			log.Error("invalid packet: " + err.Error())
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
			log.Error("write error: " + err.Error())
			sock().Close()
			upsock().Close()
			return
		}
		ping_pkt, err := mcproto.ReadPacket(conn)
		if err != nil || !ping_pkt.IsStatusPing() {
			if err == nil {
				err = errors.New("packet is not ping")
			}
			log.Error("invalid packet: " + err.Error())
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
		init_raw, err := initial_pkt.ToRawPacket()
		if err != nil {
			log.Error("Unable to encode initial packet: " + err.Error())
			return
		}
		ubufrw.Write(init_raw.ToBytes())
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
	for {
		conn, err := s.AcceptTCP()
		if err != nil {
			log.Warnf("listen_socket: error when accepting: %s", err.Error())
			continue
		}
		conn.SetNoDelay(false)
		log.Infof("listen_socket: new client %s", conn.RemoteAddr())
		// TODO: Use configurable buffer size.
		go ClientSocket(bufio.NewReadWriter(bufio.NewReaderSize(conn, 4096), bufio.NewWriterSize(conn, 4096)), func() *net.TCPConn {
			return conn
		})
	}
}

func ClientSocket(conn *bufio.ReadWriter, sock func() *net.TCPConn) {
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
			log.Error("error when reading first packet: " + err.Error())
			sock().Close()
			return
		}
	}
	handshake, err := init_pkt.ToHandShake()
	if err != nil {
		log.Error("Invalid handshake packet: " + err.Error())
		sock().Close()
		return
	}
	upstream, e := GetUpstream(handshake.ServerAddr)
	if e != nil {
		// TODO: Kick with error
		if handshake.NextState == 1 {
			log.Info("ping packet")
			pkt, err := mcproto.ReadPacket(conn)
			if err != nil {
				log.Error("Error when reading status request: " + err.Error())
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
				log.Error("Unable to make packet: " + err.Error())
				sock().Close()
				return
			}
			_, err = conn.Write(resp_pkt.ToBytes())
			if err == nil {
				err = conn.Flush()
			}
			if err != nil {
				log.Error("Unable to write response: " + err.Error())
			}
			pkt, err = mcproto.ReadPacket(conn)
			if err != nil {
				if err != io.EOF {
					log.Error("Unable to read packet: " + err.Error())
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
				log.Error("Unable to make packet: " + err.Error())
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
	go startProxy(conn, sock, upstream, handshake)
}
