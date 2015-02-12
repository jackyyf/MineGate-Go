package main

import (
	"bytes"
	"fmt"
	"github.com/jackyyf/MineGate-Go/mcchat"
	"net"
	"path"
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
var valid_chars = []byte("0123456789abcdefgijklmnopqrstuvwxyz.-")

func (upstream *Upstream) Validate() (valid bool) {
	if _, _, err := net.SplitHostPort(upstream.Server); err != nil {
		server := upstream.Server + ":25565"
		if _, _, err = net.SplitHostPort(server); err != nil {
			Fatal("Invalid upstream server: " + upstream.Server)
		}
		Infof("no port information found in %s, assume 25565", upstream.Server)
		upstream.Server = server
	}
	if upstream.ErrorMsg.Text == "" {
		Warnf("Empty error text for %s, use default string", upstream.Server)
		upstream.ErrorMsg.Text = "Connection failed to " + upstream.Server
	}
	upstream.ChatMsg = mcchat.NewMsg(upstream.ErrorMsg.Text)
	upstream.ErrorMsg.Color = strings.ToLower(upstream.ErrorMsg.Color)
	if upstream.ErrorMsg.Color != "" {
		color := mcchat.GetColor(upstream.ErrorMsg.Color)
		if color == -1 {
			Warnf("Invalid color: %s, assume white.", upstream.ErrorMsg.Color)
			upstream.ErrorMsg.Color = "white"
			color = mcchat.GetColor("white")
		}
		upstream.ChatMsg.SetColor(color)
	}
	upstream.ChatMsg.SetBold(upstream.ErrorMsg.Bold)
	upstream.ChatMsg.SetItalic(upstream.ErrorMsg.Italic)
	upstream.ChatMsg.SetUnderlined(upstream.ErrorMsg.Underlined)
	upstream.ChatMsg.SetStrikeThrough(upstream.ErrorMsg.Strikethrough)
	if upstream.ErrorMsg.Hover != "" {
		upstream.ChatMsg.HoverMsg(upstream.ErrorMsg.Hover)
	}
	if upstream.ErrorMsg.Click != "" {
		upstream.ChatMsg.ClickTarget(upstream.ErrorMsg.Click)
	}
	return true
}

func GetUpstream(hostname string) (upstream *Upstream, err error) {
	config_lock.Lock()
	defer config_lock.Unlock()
	Debugf("hostname=%s", hostname)
	for _, ch := range hostname {
		if bytes.IndexByte(valid_chars, byte(ch)) == -1 {
			return nil, fmt.Errorf("Invalid hostname: %s", hostname)
		}
	}
	for _, u := range upstreams {
		Debugf("pattern=%s", u.Pattern)
		if matched, _ := path.Match(u.Pattern, hostname); matched {
			Infof("matched server: %s", u.Server)
			return u, nil
		}
	}
	Warnf("no match for %s", hostname)
	return nil, fmt.Errorf("No such server: %s", hostname)
}
