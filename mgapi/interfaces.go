package mgapi

import (
	log "github.com/jackyyf/golog"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type MGConfig interface {
/*
	Get(string) interface{}
	GetString(string) string
	GetByteString(string) []byte
	GetInt(string) int
	GetUint(string) uint
	GetInt64(string) int64
	GetUint64(string) uint64
*/
}

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

func GetConfig() (config MGConfig) {
	return config_data
}
