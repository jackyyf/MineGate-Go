package main

import (
	"bytes"
	"fmt"
	"path"
)

type Upstream struct {
	Pattern string `yaml:"hostname"`
	Server  string `yaml:"upstream"`
	Motd    string `yaml:"motd"`
	Limit   int    `yaml:"limit"`
	Online  int    `yaml:"doesnotexist"`
}

var upstreams []Upstream
var valid_chars = []byte("0123456789abcdefgijklmnopqrstuvwxyz.-")

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
			return &u, nil
		}
	}
	Warnf("no match for %s", hostname)
	return nil, fmt.Errorf("No such server: %s", hostname)
}
