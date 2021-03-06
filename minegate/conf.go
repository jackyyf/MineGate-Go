package minegate

import (
	"errors"
	"fmt"
	mcchat "github.com/jackyyf/MineGate-Go/mcchat"
	log "github.com/jackyyf/golog"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type LogOptions struct {
	Target string `yaml:"file"`
	Level  string `yaml:"level"`
}

type Config struct {
	Log          LogOptions             `yaml:log`
	Daemonize    bool                   `yaml:"daemon"`
	Listen_addr  string                 `yaml:"listen"`
	Upstream     []*Upstream            `yaml:"upstreams"`
	NotFound     ChatMessage            `yaml:"host_not_found"`
	chatNotFound *mcchat.ChatMsg        `yaml:"-"`
	Extras       map[string]interface{} `yaml:",inline"`
}

var config Config

var config_file, _ = filepath.Abs("./config.yml")

var config_lock sync.Mutex

func ToChatMsg(msg *ChatMessage) (res *mcchat.ChatMsg) {
	res = mcchat.NewMsg(msg.Text)
	msg.Color = strings.ToLower(msg.Color)
	if msg.Color != "" {
		color := mcchat.GetColor(msg.Color)
		if color == -1 {
			log.Warnf("Invalid color: %s, assume white.", msg.Color)
			msg.Color = "white"
			color = mcchat.GetColor("white")
		}
		res.SetColor(color)
	}
	res.SetBold(msg.Bold)
	res.SetItalic(msg.Italic)
	res.SetUnderlined(msg.Underlined)
	res.SetStrikeThrough(msg.Strikethrough)
	if msg.Hover != "" {
		res.HoverMsg(msg.Hover)
	}
	if msg.Click != "" {
		res.ClickTarget(msg.Click)
	}
	return
}

func SetConfig(conf string) {
	log.Infof("using config file %s", conf)
	config_file = conf
}

func GetExtraConf(path string) (val interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Recovered panic: config.GetExtra(%s), err=%+v", path, r)
			log.Debugf("Upstream: %+v", config)
			val = nil
			err = errors.New("panic when getting config.")
			return
		}
	}()
	paths := strings.Split(path, ".")
	cur := reflect.ValueOf(config.Extras)
	// ROOT can't be an array, so assume no config path starts with #
	for _, path := range paths {
		index := strings.Split(path, "#")
		prefix, index := index[0], index[1:]
		if cur.Kind() != reflect.Map {
			log.Warnf("config.GetExtra(%s): unable to fetch key %s, not a map.", path, prefix)
			return nil, fmt.Errorf("index key on non-map type")
		}
		cur = reflect.ValueOf(cur.MapIndex(reflect.ValueOf(prefix)).Interface())
		for _, idx := range index {
			i, err := strconv.ParseInt(idx, 0, 0)
			if err != nil {
				log.Errorf("config.GetExtra(%s): unable to parse %s: %s", path, idx, err.Error())
				return nil, fmt.Errorf("Unable to parse %s: %s", idx, err.Error())
			}
			if cur.Kind() != reflect.Slice {
				log.Warnf("config.GetExtra(%s): unable to index value, not a slice", path)
				return nil, errors.New("Unable to index value, not a slice.")
			}
			if int(i) >= cur.Len() {
				log.Warnf("config.GetExtra(%s): index %d out of range", path, i)
				return nil, errors.New("Index out of range.")
			}
			cur = reflect.ValueOf(cur.Index(int(i)).Interface())
		}
	}
	return cur.Interface(), nil
}

func validateConfig() {
	invalid_upstreams := make([]int, 0, len(config.Upstream))
	for idx, upstream := range config.Upstream {
		if !upstream.Validate() {
			log.Errorf("Upstream %s is not activated.", upstream.Server)
			invalid_upstreams = append(invalid_upstreams, idx)
		}
	}
	for delta, idx := range invalid_upstreams {
		idx -= delta
		config.Upstream[idx] = nil
		config.Upstream = append(config.Upstream[:idx], config.Upstream[idx+1:]...)
	}
	if config.NotFound.Text == "" {
		log.Warn("Empty error text for not found error, use default string")
		config.NotFound.Text = "No such host."
	}
	config.chatNotFound = ToChatMsg(&config.NotFound)
}

func confInit() {
	content, err := ioutil.ReadFile(config_file)
	if err != nil {
		log.Fatalf("unable to load config %s: %s", config_file, err.Error())
	}
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		log.Fatalf("error when parsing config file %s: %s", config_file, err.Error())
	}
	validateConfig()
	if config.Log.Target != "" && config.Log.Target != "-" {
		config.Log.Target, _ = filepath.Abs(config.Log.Target)
		log.Info("log path: " + config.Log.Target)
	}
	log.Stop()
	if config.Daemonize {
		Daemonize()
	}
	log.Start()
	if config.Log.Level != "" {
		level := log.ToLevel(config.Log.Level)
		if level == log.INVALID {
			log.Errorf("Invalid log level %s", config.Log.Level)
		} else {
			log.SetLogLevel(level)
		}
	}
	if config.Log.Target != "" && config.Log.Target != "-" {
		err := log.Open(config.Log.Target)
		if err != nil {
			log.Fatalf("Unable to open log %s: %s", config.Log.Target, err.Error())
		}
	}
	log.Info("config loaded.")
	log.Info("server listen on: " + config.Listen_addr)
	log.Infof("%d upstream server(s) found", len(config.Upstream))
}

func ConfReload() {
	// Do not panic!
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("paniced when reloading config %s, recovered.", config_file)
			log.Errorf("panic: %s", r)
		}
	}()
	log.Warn("Reloading config")
	content, err := ioutil.ReadFile(config_file)
	if err != nil {
		log.Errorf("unable to reload config %s: %s", config_file, err.Error())
		return
	}
	prev_listen := config.Listen_addr
	config_lock.Lock()
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		log.Errorf("error when parsing config file %s: %s", config_file, err.Error())
		return
	}
	validateConfig()
	config_lock.Unlock()
	log.Info("config reloaded.")
	if config.Listen_addr != prev_listen {
		log.Warnf("config reload will not reopen server socket, thus no effect on listen address")
	}
	log.Infof("%d upstream server(s) found", len(config.Upstream))
}
