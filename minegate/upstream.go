package main

import (
	"bytes"
	"fmt"
	"github.com/jackyyf/MineGate-Go/mcchat"
	log "github.com/jackyyf/golog"
	"net"
	"path"
	"strconv"
	"strings"
)

type ChatMessage struct {
	Text          string
	Color         string
	Bold          bool
	Italic        bool
	Underlined    bool
	Strikethrough bool
	Hover         string
	Click         string
}

type Upstream struct {
	Pattern  string          `yaml:"hostname"`
	Server   string          `yaml:"upstream"`
	ErrorMsg ChatMessage     `yaml:"onerror"`
	ChatMsg  *mcchat.ChatMsg `yaml:"-"`
}

var upstreams []*Upstream
var valid_host = []byte("0123456789abcdefgijklmnopqrstuvwxyz.-")
var valid_pattern = []byte("0123456789abcdefgijklmnopqrstuvwxyz.-*?")

func (upstream *Upstream) Validate() (valid bool) {
	var host, port string
	host, port, err := net.SplitHostPort(upstream.Server)
	if err != nil {
		server := upstream.Server + ":25565"
		if host, port, err = net.SplitHostPort(server); err != nil {
			log.Error("Invalid upstream server: " + upstream.Server)
			return false
		}
		log.Infof("no port information found in %s, assume 25565", upstream.Server)
		upstream.Server = server
	}
	p, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		log.Errorf("Invalid port %s: %s", port, err.Error())
		return false
	}
	host = strings.ToLower(host)
	if !checkHost(host) {
		log.Error("Invalid upstream host: " + host)
		return false
	}
	upstream.Server = net.JoinHostPort(host, fmt.Sprintf("%d", p))
	upstream.Pattern = strings.ToLower(upstream.Pattern)
	if !checkPattern(upstream.Pattern) {
		log.Error("Invalid pattern: " + upstream.Pattern)
		return false
	}
	if upstream.ErrorMsg.Text == "" {
		log.Warnf("Empty error text for %s, use default string", upstream.Server)
		upstream.ErrorMsg.Text = "Connection failed to " + upstream.Server
	}
	upstream.ChatMsg = ToChatMsg(&upstream.ErrorMsg)
	return true
}

func checkHost(host string) (valid bool) {
	for _, ch := range host {
		if bytes.IndexByte(valid_host, byte(ch)) == -1 {
			return false
		}
	}
	return true
}

func checkPattern(pattern string) (valid bool) {
	for _, ch := range pattern {
		if bytes.IndexByte(valid_pattern, byte(ch)) == -1 {
			return false
		}
	}
	return true
}

func GetUpstream(hostname string) (upstream *Upstream, err *mcchat.ChatMsg) {
	config_lock.Lock()
	defer config_lock.Unlock()
	log.Debugf("hostname=%s", hostname)
	if !checkHost(hostname) {
		return nil, config.chatBadHost
	}
	for _, u := range upstreams {
		log.Debugf("pattern=%s", u.Pattern)
		if matched, _ := path.Match(u.Pattern, hostname); matched {
			log.Infof("matched server: %s", u.Server)
			return u, nil
		}
	}
	log.Warnf("no match for %s", hostname)
	return nil, config.chatNotFound
}
