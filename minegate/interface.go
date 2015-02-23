package minegate

import (
	"errors"
	"fmt"
	"github.com/jackyyf/MineGate-Go/mcproto"
	log "github.com/jackyyf/golog"
	"net"
)

type NetworkEvent struct {
	RemoteAddr net.Addr
	connID     uintptr
}

type PostAcceptEvent struct {
	NetworkEvent
	reject bool
}

type PreRoutingEvent struct {
	NetworkEvent
	Packet *mcproto.MCHandShake
}

type PingRequestEvent struct {
	NetworkEvent
	Packet *mcproto.MCHandShake
}

type LoginRequestEvent struct {
	NetworkEvent
	InitPacket  *mcproto.MCHandShake
	LoginPacket *mcproto.MCLogin
}

type PreStatusResponseEvent struct {
	NetworkEvent
	Packet *mcproto.MCStatusResponse
}

/*

type PostCloseEvent struct {
	NetworkEvent
}

*/

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

func (event *PostAcceptEvent) Rejected() (reject bool) {
	return event.reject
}

func (event *PostAcceptEvent) Allow() {
	event.reject = false
}

func (event *PostAcceptEvent) Reject() {
	event.reject = true
}

// This file defines all model interfaces.

type PreLoadConfigFunc func()
type PostLoadConfigFunc func()
type PostAcceptFunc func(*PostAcceptEvent)
type PreRoutingFunc func(*PreRoutingEvent)
type PingRequestFunc func(*PingRequestEvent)
type LoginRequestFunc func(*LoginRequestEvent)
type PreStatusResponseFunc func(*PreStatusResponseEvent)

// type PostCloseFunc func(*PostCloseEvent) // Not implemented

type preLoadConfigHandler []PreLoadConfigFunc
type postLoadConfigHandler []PostLoadConfigFunc
type postAcceptHandler []PostAcceptFunc
type preRoutingHandler []PreRoutingFunc
type pingRequestHandler []PingRequestFunc
type loginRequestHandler []LoginRequestFunc
type preStatusResponseHandler []PreStatusResponseFunc

// type postCloseHandler []PostCloseFunc // Not implemented

var preLoadConfigHandlers [40]preLoadConfigHandler
var postLoadConfigHandlers [40]postLoadConfigHandler
var postAcceptHandlers [40]postAcceptHandler
var preRoutingHandlers [40]preRoutingHandler
var pingRequestHandlers [40]pingRequestHandler
var loginRequestHandlers [40]loginRequestHandler
var preStatusResponseHandlers [40]preStatusResponseHandler

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
		log.Infof("Calling PostAccept priority=%d", p)
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
		log.Infof("Calling PreRouting priority=%d", p)
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
		log.Infof("Calling PingRequest priority=%d", p)
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
		log.Infof("Calling PingRequest priority=%d", p)
		for _, handler := range l {
			handler(event)
		}
	}
}
