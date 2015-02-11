package main

import (
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"sync"
)

type Config struct {
	Listen_addr string     "listen"
	Upstream    []Upstream "upstreams"
}

var config Config

var config_file = "./config.yml"

var config_lock sync.Mutex

func SetConfig(conf string) {
	Infof("using config file %s", conf)
	config_file = conf
}

func ConfInit() {
	Debug("call: ConfInit")
	content, err := ioutil.ReadFile(config_file)
	if err != nil {
		Fatalf("unable to load config %s: %s", config_file, err.Error())
	}
	config_lock.Lock()
	defer config_lock.Unlock()
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		Fatalf("error when parsing config file %s: %s", config_file, err.Error())
	}
	Info("config loaded.")
	Info("server listen on: " + config.Listen_addr)
	Infof("%d upstream server(s) found", len(config.Upstream))
	upstreams = config.Upstream
}

func ConfReload() {
	// Do not panic!
	defer func() {
		if r := recover(); r != nil {
			Errorf("paniced when reloading config %s, recovered.", config_file)
			Errorf("panic: %s", r)
		}
	}()
	Warn("Reloading config")
	content, err := ioutil.ReadFile(config_file)
	if err != nil {
		Errorf("unable to reload config %s: %s", config_file, err.Error())
	}
	prev_listen := config.Listen_addr
	config_lock.Lock()
	defer config_lock.Unlock()
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		Errorf("error when parsing config file %s: %s", config_file, err.Error())
	}
	Info("config reloaded.")
	if config.Listen_addr != prev_listen {
		Warnf("config reload will not reopen server socket, thus no effect on listen address")
	}
	Infof("%d upstream server(s) found", len(config.Upstream))
	upstreams = config.Upstream
}
