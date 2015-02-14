package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestInvalidHost(t *testing.T) {
	if err := ioutil.WriteFile("invalid_host.yml", []byte(
		`
listen: '[::]:25565'
upstreams:
  -
    hostname: server1.local
    upstream: '-_-:invalidhost!:25568'
  -
    hostname: '*.local'
    upstream: 127.0.0.1:25566
  -
    hostname: '*'
    upstream: '-_-:anotherinvalid!!!'
  -
    hostname: '233'
    upstream: '1.2.3.4'`), 0644); err != nil {
		t.Fatal("Unable to write to invalid_host.yml")
		return
	}
	SetConfig("invalid_host.yml")
	ConfReload()
	ulen := len(config.Upstream)
	if len(config.Upstream) != 2 {
		t.Errorf("There should be 2 valid upstreams, %d found", ulen)
	} else {
		t.Log("Ok. 2 valid upstreams")
	}
	if ulen > 0 {
		if config.Upstream[0].Server != "127.0.0.1:25566" {
			t.Errorf("Upstream 0 should be 127.0.0.1:25566, %s found", config.Upstream[0].Server)
		} else {
			t.Log("Ok. upstream 0 is 127.0.0.1:25566")
		}
		if ulen > 1 {
			if config.Upstream[1].Server != "1.2.3.4:25565" {
				t.Errorf("Upstream 1 should be 1.2.3.4:25565, %s found", config.Upstream[1].Server)
			} else {
				t.Log("Ok. upstream 1 is 1.2.3.4:25565")
			}
		}
	}
	os.Remove("invalid_host.yml")
}
