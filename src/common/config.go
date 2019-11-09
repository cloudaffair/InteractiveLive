package common

import (
	"encoding/json"
	"os"
	"io/ioutil"
	"sync"
	"fmt"
)

const localConfigFilePath       = "dev_config.json"
var _config_init_ctx sync.Once
var _config_instance *Config
var _config_error error


type Config struct {
	Dependencies    map[string]interface{} `json:"dependencies"`
}

func GetConfig() (*Config, error) {
	_config_init_ctx.Do(func() {
		var err error
		fmt.Println("Loading config" )
		_config_instance, err = loadConfig()
		if err != nil {
			_config_error = err
		}
	})
	return _config_instance, _config_error
}

func loadConfig() (*Config, error) {
	projectRoot := os.Getenv("PWD")
	configFilePath := localConfigFilePath
	if len(projectRoot) > 0 {
		configFilePath = projectRoot + "/" + localConfigFilePath
	}
	fmt.Println("Trying configuration", configFilePath)
	data, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, Errorf("Error reading json from config file. Error: %s\nRaw data: %v\n", err.Error(), string(data))
	}
	s := &Config{}
	err = json.Unmarshal(data, s)

	if err != nil {
		return nil, Errorf("Error unmarshalling json from config file. Error: %s\nRaw data: %v\n", err.Error(), string(data))
	}

	return s, nil
}