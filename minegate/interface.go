package minegate

import (
	"errors"
	"fmt"
	"github.com/jackyyf/MineGate-Go/mcproto"
	log "github.com/jackyyf/golog"
	"net"
)

type NetworkEvent struct {
	RemoteAddr *net.TCPAddr
	connID     uintptr
	log_prefix string
}

type RejectPoint struct {
	reject bool
	reason string
}

type PostAcceptEvent struct {
	NetworkEvent
	RejectPoint
}

type PreRoutingEvent struct {
	NetworkEvent
	RejectPoint
	Packet *mcproto.MCHandShake
}

type PingRequestEvent struct {
	NetworkEvent
	RejectPoint
	Packet   *mcproto.MCHandShake
	Upstream *Upstream
}

type LoginRequestEvent struct {
	NetworkEvent
	RejectPoint
	InitPacket  *mcproto.MCHandShake
	LoginPacket *mcproto.MCLogin
	Upstream    *Upstream
}

type StartProxyEvent struct {
	NetworkEvent
	InitPacket  *mcproto.MCHandShake
	LoginPacket *mcproto.MCLogin
	Upstream    *Upstream
}

type PreStatusResponseEvent struct {
	NetworkEvent
	Packet   *mcproto.MCStatusResponse
	Upstream *Upstream
}

type DisconnectEvent struct {
	NetworkEvent
}

func (event *NetworkEvent) GetRemoteIP() (ip string) {
	addr, _, err := net.SplitHostPort(event.RemoteAddr.String())
	if err != nil {
		return ""
	}
	return addr
}

func (event *NetworkEvent) GetConnID() (connID uintptr) {
	return event.connID
}

func (event *NetworkEvent) Debugf(format string, v ...interface{}) {
	if event.log_prefix == "" {
		event.log_prefix = fmt.Sprintf("[#%d %s]", event.connID, event.RemoteAddr)
	}
	log.Debugf(event.log_prefix+format, v...)
}

func (event *NetworkEvent) Infof(format string, v ...interface{}) {
	if event.log_prefix == "" {
		event.log_prefix = fmt.Sprintf("[#%d %s]", event.connID, event.RemoteAddr)
	}
	log.Infof(event.log_prefix+format, v...)
}

func (event *NetworkEvent) Warnf(format string, v ...interface{}) {
	if event.log_prefix == "" {
		event.log_prefix = fmt.Sprintf("[#%d %s]", event.connID, event.RemoteAddr)
	}
	log.Warnf(event.log_prefix+format, v...)
}

func (event *NetworkEvent) Errorf(format string, v ...interface{}) {
	if event.log_prefix == "" {
		event.log_prefix = fmt.Sprintf("[#%d %s]", event.connID, event.RemoteAddr)
	}
	log.Errorf(event.log_prefix+format, v...)
}

func (event *NetworkEvent) Fatalf(format string, v ...interface{}) {
	if event.log_prefix == "" {
		event.log_prefix = fmt.Sprintf("[#%d %s]", event.connID, event.RemoteAddr)
	}
	log.Fatalf(event.log_prefix+format, v...)
}

func (event *RejectPoint) Rejected() (reject bool) {
	return event.reject
}

func (event *RejectPoint) Allow() {
	event.reject = false
}

func (event *RejectPoint) Reject() {
	event.reject = true
}

func (event *RejectPoint) Reason(reason string) {
	event.reject = true
	event.reason = reason
}

// This file defines all model interfaces.

type PreLoadConfigFunc func()
type PostLoadConfigFunc func()
type PostAcceptFunc func(*PostAcceptEvent)
type PreRoutingFunc func(*PreRoutingEvent)
type PingRequestFunc func(*PingRequestEvent)
type LoginRequestFunc func(*LoginRequestEvent)
type StartProxyFunc func(*StartProxyEvent)
type PreStatusResponseFunc func(*PreStatusResponseEvent)
type DisconnectFunc func(*DisconnectEvent)

// type PostCloseFunc func(*PostCloseEvent) // Not implemented

type preLoadConfigHandler []PreLoadConfigFunc
type postLoadConfigHandler []PostLoadConfigFunc
type postAcceptHandler []PostAcceptFunc
type preRoutingHandler []PreRoutingFunc
type pingRequestHandler []PingRequestFunc
type loginRequestHandler []LoginRequestFunc
type startProxyHandler []StartProxyFunc
type preStatusResponseHandler []PreStatusResponseFunc
type disconnectHandler []DisconnectFunc

// type postCloseHandler []PostCloseFunc // Not implemented

var preLoadConfigHandlers [40]preLoadConfigHandler
var postLoadConfigHandlers [40]postLoadConfigHandler
var postAcceptHandlers [40]postAcceptHandler
var preRoutingHandlers [40]preRoutingHandler
var pingRequestHandlers [40]pingRequestHandler
var loginRequestHandlers [40]loginRequestHandler
var startProxyHandlers [40]startProxyHandler
var preStatusResponseHandlers [40]preStatusResponseHandler
var disconnectHandlers [40]disconnectHandler

// var postCloseHandlers [40]postCloseFuncHandler // Not implemented

func OnPreLoadConfig(handle PreLoadConfigFunc, priority int) (err error) {
	if priority < 0 || priority > 39 {
		log.Errorf("Invalid priority %d: not in range [0, 39]", priority)
		return fmt.Errorf("priority check failure: %d not in range [0, 39]", priority)
	}
	if handle == nil {
		log.Error("Attempt to register nil handler")
		return errors.New("Nil handler!")
	}
	if preLoadConfigHandlers[priority] == nil {
		preLoadConfigHandlers[priority] = make(preLoadConfigHandler, 0, 16)
	}
	preLoadConfigHandlers[priority] = append(preLoadConfigHandlers[priority], handle)
	log.Infof("Registered preLoadConfig handler at priority %d", priority)
	return nil
}

func OnPostLoadConfig(handle PostLoadConfigFunc, priority int) (err error) {
	if priority < 0 || priority > 39 {
		log.Errorf("Invalid priority %d: not in range [0, 39]", priority)
		return fmt.Errorf("priority check failure: %d not in range [0, 39]", priority)
	}
	if handle == nil {
		log.Error("Attempt to register nil handler")
		return errors.New("Nil handler!")
	}
	if postLoadConfigHandlers[priority] == nil {
		postLoadConfigHandlers[priority] = make(postLoadConfigHandler, 0, 16)
	}
	postLoadConfigHandlers[priority] = append(postLoadConfigHandlers[priority], handle)
	log.Infof("Registered postLoadConfig handler at priority %d", priority)
	return nil
}

func OnPostAccept(handle PostAcceptFunc, priority int) (err error) {
	if priority < 0 || priority > 39 {
		log.Errorf("Invalid priority %d: not in range [0, 39]", priority)
		return fmt.Errorf("priority check failure: %d not in range [0, 39]", priority)
	}
	if handle == nil {
		log.Error("Attempt to register nil handler")
		return errors.New("Nil handler!")
	}
	if postAcceptHandlers[priority] == nil {
		postAcceptHandlers[priority] = make(postAcceptHandler, 0, 16)
	}
	postAcceptHandlers[priority] = append(postAcceptHandlers[priority], handle)
	log.Infof("Registered postAccept handler at priority %d", priority)
	return nil
}

func OnPreRouting(handle PreRoutingFunc, priority int) (err error) {
	if priority < 0 || priority > 39 {
		log.Errorf("Invalid priority %d: not in range [0, 39]", priority)
		return fmt.Errorf("priority check failure: %d not in range [0, 39]", priority)
	}
	if handle == nil {
		log.Error("Attempt to register nil handler")
		return errors.New("Nil handler!")
	}
	if preRoutingHandlers[priority] == nil {
		preRoutingHandlers[priority] = make(preRoutingHandler, 0, 16)
	}
	preRoutingHandlers[priority] = append(preRoutingHandlers[priority], handle)
	log.Infof("Registered preRouting handler at priority %d", priority)
	return nil
}

func OnPingRequest(handle PingRequestFunc, priority int) (err error) {
	if priority < 0 || priority > 39 {
		log.Errorf("Invalid priority %d: not in range [0, 39]", priority)
		return fmt.Errorf("priority check failure: %d not in range [0, 39]", priority)
	}
	if handle == nil {
		log.Error("Attempt to register nil handler")
		return errors.New("Nil handler!")
	}
	if pingRequestHandlers[priority] == nil {
		pingRequestHandlers[priority] = make(pingRequestHandler, 0, 16)
	}
	pingRequestHandlers[priority] = append(pingRequestHandlers[priority], handle)
	log.Infof("Registered pingRequest handler at priority %d", priority)
	return nil
}

func OnLoginRequest(handle LoginRequestFunc, priority int) (err error) {
	if priority < 0 || priority > 39 {
		log.Errorf("Invalid priority %d: not in range [0, 39]", priority)
		return fmt.Errorf("priority check failure: %d not in range [0, 39]", priority)
	}
	if handle == nil {
		log.Error("Attempt to register nil handler")
		return errors.New("Nil handler!")
	}
	if loginRequestHandlers[priority] == nil {
		loginRequestHandlers[priority] = make(loginRequestHandler, 0, 16)
	}
	loginRequestHandlers[priority] = append(loginRequestHandlers[priority], handle)
	log.Infof("Registered pingRequest handler at priority %d", priority)
	return nil
}

func OnStartProxy(handle StartProxyFunc, priority int) (err error) {
	if priority < 0 || priority > 39 {
		log.Errorf("Invalid priority %d: not in range [0, 39]", priority)
		return fmt.Errorf("priority check failure: %d not in range [0, 39]", priority)
	}
	if handle == nil {
		log.Error("Attempt to register nil handler")
		return errors.New("Nil handler!")
	}
	if startProxyHandlers[priority] == nil {
		startProxyHandlers[priority] = make(startProxyHandler, 0, 16)
	}
	startProxyHandlers[priority] = append(startProxyHandlers[priority], handle)
	log.Infof("Registered startProxy handler at priority %d", priority)
	return nil
}

func OnPreStatusResponse(handle PreStatusResponseFunc, priority int) (err error) {
	if priority < 0 || priority > 39 {
		log.Errorf("Invalid priority %d: not in range [0, 39]", priority)
		return fmt.Errorf("priority check failure: %d not in range [0, 39]", priority)
	}
	if handle == nil {
		log.Error("Attempt to register nil handler")
		return errors.New("Nil handler!")
	}
	if preStatusResponseHandlers[priority] == nil {
		preStatusResponseHandlers[priority] = make(preStatusResponseHandler, 0, 16)
	}
	preStatusResponseHandlers[priority] = append(preStatusResponseHandlers[priority], handle)
	log.Infof("Registered preStatusResponse handler at priority %d", priority)
	return nil
}

func OnDisconnect(handle DisconnectFunc, priority int) (err error) {
	if priority < 0 || priority > 39 {
		log.Errorf("Invalid priority %d: not in range [0, 39]", priority)
		return fmt.Errorf("priority check failure: %d not in range [0, 39]", priority)
	}
	if handle == nil {
		log.Error("Attempt to register nil handler")
		return errors.New("Nil handler!")
	}
	if disconnectHandlers[priority] == nil {
		disconnectHandlers[priority] = make(disconnectHandler, 0, 16)
	}
	disconnectHandlers[priority] = append(disconnectHandlers[priority], handle)
	log.Infof("Registered disconnect handler at priority %d", priority)
	return nil
}

func PreLoadConfig() {
	for p, l := range preLoadConfigHandlers {
		if l == nil {
			continue
		}
		log.Infof("Calling PreLoadConfig priority=%d", p)
		for _, handler := range l {
			handler()
		}
	}
}

func PostLoadConfig() {
	for p, l := range postLoadConfigHandlers {
		if l == nil {
			continue
		}
		log.Infof("Calling PostLoadConfig priority=%d", p)
		for _, handler := range l {
			handler()
		}
	}
}

func PostAccept(event *PostAcceptEvent) {
	for p, l := range postAcceptHandlers {
		if l == nil {
			continue
		}
		event.Infof("Calling PostAccept priority=%d", p)
		for _, handler := range l {
			handler(event)
		}
	}
}

func PreRouting(event *PreRoutingEvent) {
	for p, l := range preRoutingHandlers {
		if l == nil {
			continue
		}
		event.Infof("Calling PreRouting priority=%d", p)
		for _, handler := range l {
			handler(event)
		}
	}
}

func PingRequest(event *PingRequestEvent) {
	for p, l := range pingRequestHandlers {
		if l == nil {
			continue
		}
		event.Infof("Calling PingRequest priority=%d", p)
		for _, handler := range l {
			handler(event)
		}
	}
}

func PreStatusResponse(event *PreStatusResponseEvent) {
	for p, l := range preStatusResponseHandlers {
		if l == nil {
			continue
		}
		event.Infof("Calling PreStatusResponse priority=%d", p)
		for _, handler := range l {
			handler(event)
		}
	}
}

func StartProxy(event *StartProxyEvent) {
	for p, l := range startProxyHandlers {
		if l == nil {
			continue
		}
		event.Infof("Calling StartProxy priority=%d", p)
		for _, handler := range l {
			handler(event)
		}
	}
}

func LoginRequest(event *LoginRequestEvent) {
	for p, l := range loginRequestHandlers {
		if l == nil {
			continue
		}
		event.Infof("Calling PingRequest priority=%d", p)
		for _, handler := range l {
			handler(event)
		}
	}
}

func Disconnect(event *DisconnectEvent) {
	for p, l := range disconnectHandlers {
		if l == nil {
			continue
		}
		event.Infof("Calling Disconnect priority=%d", p)
		for _, handler := range l {
			handler(event)
		}
	}
}
