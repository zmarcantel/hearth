package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

const Name string = ".hearthrc"

func Path() string {
	return path.Join(os.Getenv("HOME"), Name)
}

// Opens the ~/.hearthrc file. This fille cannot change nor be moved
// so this function is really a convenience function
func Open() (Config, error) {
	var config Config
	default_file := Path()
	if strings.HasPrefix(default_file, "~/") {
		default_file = path.Join(os.Getenv("HOME"), default_file[2:])
	}

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

// Creates a new config inside the given repo_path.
// While it may be tempting to use this to create a config anywhere you want,
// the de facto usage requires it to be made inside a repo
func Create(repo_path string) (Config, error) {
	var config Config
	config.BaseDirectory = repo_path
	conf_path := path.Join(repo_path, Name)

	if _, err := os.Stat(conf_path); err == nil {
		return config, fmt.Errorf("%s already exists, and will not overwrite.", conf_path)
	}

	// write the config file out
	return config, config.Write(conf_path)
}

// Write the config file to the given path
func (c Config) Write(path string) error {
	config_bytes, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("could not marshal new config: %s", err.Error())
	}

	err = ioutil.WriteFile(path, config_bytes, 0755)
	if err != nil {
		return fmt.Errorf("could not save new config: %s", err.Error())
	}

	return nil
}
