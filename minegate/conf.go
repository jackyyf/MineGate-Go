package main

import (
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Listen_addr string "listen"
	Upstream []Upstream "upstreams"
}

var config Config

var config_file = "./config.yml"

func SetConfig(conf string) {
	Infof("using config file %s", conf)
	config_file = conf
}

func ConfInit() {
	Debug("call: ConfInit")
	content, err := ioutil.ReadFile(config_file)
	if err != nil {
		Fatalf("unable to load config %s: %s", config_file, err)
	}
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		Fatalf("error when parsing config file %s: %s", config_file, err)
	}
	Info("config loaded.")
	Info("server listen on: " + config.Listen_addr)
	Infof("%d upstream server(s) found", len(config.Upstream))
	upstreams = config.Upstream
}
