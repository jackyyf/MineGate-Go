package minegate

import (
	log "github.com/jackyyf/golog"
	"io/ioutil"
	"os"
	"testing"
)

func TestEmptyErrorMsg(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Recovered from panic: %s", r)
			return
		}
	}()
	defer os.Remove("empty_msg.yml")
	log.SetLogLevel(log.FATAL)
	if err := ioutil.WriteFile("empty_msg.yml", []byte(
		`
listen: '[::]:23333'
upstreams:
  - hostname: server.local
    upstream: test.local:23345`), 0644); err != nil {
		t.Fatal("Unable to write to invalid_host.yml")
		return
	}
	SetConfig("empty_msg.yml")
	confInit()
	t.Log("Everything fine :)")
}

func TestInvalidHost(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Recovered from panic: %s", r)
			return
		}
	}()
	defer os.Remove("invalid_host.yml")
	log.SetLogLevel(log.FATAL)
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
	confInit()
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
}

func TestPortRange(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Recovered from panic: %s", r)
			return
		}
	}()
	defer os.Remove("port_range.yml")
	log.SetLogLevel(log.FATAL)
	if err := ioutil.WriteFile("port_range.yml", []byte(
		`
listen: '1:25565'
upstreams:
- hostname: server.local
  upstream: localhost:65537
- hostname: server2.local
  upstream: localhost:-1`), 0644); err != nil {
		t.Fatal("Unable to write to port_range.yml")
		return
	}
	SetConfig("port_range.yml")
	confInit()
	if len(config.Upstream) != 0 {
		t.Fatalf("No valid upstreams provided, but %d upstreams found!", len(config.Upstream))
		return
	}
	t.Log("Ok, no valid upstream.")
}
