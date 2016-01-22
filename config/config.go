package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	yaml "gopkg.in/yaml.v2"
)

const Name string = ".hearthrc"

func Path() string {
	return path.Join(os.Getenv("HOME"), Name)
}

// Opens the ~/.hearthrc file.
// If this file does not exist, an interactive prompt begins that will attempt to create
// one. Because this function is integral to startup, we pass back a return code.
func Open() (Config, error) {
	var config Config
	default_file := Path()

	// try to read the default
	config_bytes, err := ioutil.ReadFile(default_file)
	if err != nil {
		return Config{}, err
	}

	// make a config out of it
	if err = yaml.Unmarshal(config_bytes, &config); err != nil {
		return config, fmt.Errorf("failed to parse hearthrc: %s", err.Error())
	}

	return config, nil
}

func Create(conf_path, repo_path string) (Config, error) {
	var config Config
	config.BaseDirectory = repo_path

	if _, err := os.Stat(conf_path); err == nil {
		return config, fmt.Errorf("%s already exists, and will not overwrite.", conf_path)
	}

	// write the config file out
	return config, Write(conf_path, config)
}

func Write(path string, conf Config) error {
	config_bytes, err := yaml.Marshal(conf)
	if err != nil {
		return fmt.Errorf("could not marshal new config: %s", err.Error())
	}

	err = ioutil.WriteFile(path, config_bytes, 0755)
	if err != nil {
		return fmt.Errorf("could not save new config: %s", err.Error())
	}

	return nil
}

func Load(path string) (Config, error) {
	// check if we have a config file
	config_bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var config Config
	return config, yaml.Unmarshal(config_bytes, &config)
}
