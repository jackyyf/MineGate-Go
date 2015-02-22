package mgapi

import (
	"errors"
	// "fmt"
	log "github.com/jackyyf/golog"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

type config map[string]interface{}

var config_file string
var config_data *config

func SetConfigFile(file string) {
	config_file = file
}

func LoadConfig() (err error) {
	log.Debug("mgapi.LoadConfig")
	data, err := ioutil.ReadFile(config_file)
	if err != nil {
		return
	}
	var new_config *config
	err = yaml.Unmarshal(data, new_config)
	if err != nil {
		return
	}
	config_data = new_config
	return nil
}

func (conf *config) Get(str string) (val interface{}, err error) {
	if conf == nil {
		return nil, errors.New("Config is not initialized.")
	}
	paths := strings.Split(str, ".")
	// ROOT can't be an array, so assume no config path starts with #
	for _, path := range paths {
		index := strings.Split(path, "#")
		prefix, index := index[0], index[1:]
	}
	return
}
