package minegate

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/jackyyf/MineGate-Go/mcchat"
	log "github.com/jackyyf/golog"
	"net"
	"path"
	"reflect"
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
	Pattern  string                 `yaml:"hostname"`
	Server   string                 `yaml:"upstream"`
	ErrorMsg ChatMessage            `yaml:"onerror"`
	ChatMsg  *mcchat.ChatMsg        `yaml:"-"`
	Extras   map[string]interface{} `yaml:",inline"`
}

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
	if !CheckHost(host) {
		log.Error("Invalid upstream host: " + host)
		return false
	}
	upstream.Server = net.JoinHostPort(host, fmt.Sprintf("%d", p))
	upstream.Pattern = strings.ToLower(upstream.Pattern)
	if !CheckPattern(upstream.Pattern) {
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

func (upstream *Upstream) GetExtra(path string) (val interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Recovered panic: upstream.GetExtra(%s), err=%+v", path, r)
			log.Debugf("Upstream: %+v", upstream)
			val = nil
			err = errors.New("panic when getting config.")
			return
		}
	}()
	paths := strings.Split(path, ".")
	cur := reflect.ValueOf(upstream.Extras)
	// ROOT can't be an array, so assume no config path starts with #
	for _, path := range paths {
		index := strings.Split(path, "#")
		prefix, index := index[0], index[1:]
		if cur.Kind() != reflect.Map {
			log.Warnf("upstream.GetExtra(%s): unable to fetch key %s, not a map.", path, prefix)
			return nil, fmt.Errorf("index key on non-map type")
		}
		cur = reflect.ValueOf(cur.MapIndex(reflect.ValueOf(prefix)).Interface())
		for _, idx := range index {
			i, err := strconv.ParseInt(idx, 0, 0)
			if err != nil {
				log.Errorf("upstream.GetExtra(%s): unable to parse %s: %s", path, idx, err.Error())
				return nil, fmt.Errorf("Unable to parse %s: %s", idx, err.Error())
			}
			if cur.Kind() != reflect.Slice {
				log.Warnf("upstream.GetExtra(%s): unable to index value, not a slice", path)
				return nil, errors.New("Unable to index value, not a slice.")
			}
			if int(i) >= cur.Len() {
				log.Warnf("upstream.GetExtra(%s): index %d out of range", path, i)
				return nil, errors.New("Index out of range.")
			}
			cur = reflect.ValueOf(cur.Index(int(i)).Interface())
		}
	}
	return cur.Interface(), nil
}

func CheckHost(host string) (valid bool) {
	for _, ch := range host {
		if bytes.IndexByte(valid_host, byte(ch)) == -1 {
			return false
		}
	}
	return true
}

func CheckPattern(pattern string) (valid bool) {
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
	hostname = strings.ToLower(hostname)
	if !CheckHost(hostname) {
		return nil, config.chatBadHost
	}
	for _, u := range config.Upstream {
		log.Debugf("pattern=%s", u.Pattern)
		if matched, _ := path.Match(u.Pattern, hostname); matched {
			log.Infof("matched server: %s", u.Server)
			return u, nil
		}
	}
	log.Warnf("no match for %s", hostname)
	return nil, config.chatNotFound
}
