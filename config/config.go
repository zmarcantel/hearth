package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

// Opens the ~/.hearthrc file.
// If this file does not exist, an interactive prompt begins that will attempt to create
// one. Because this function is integral to startup, we pass back a return code.
func Open() (Config, error) {
	var config Config
	default_file := path.Join(os.Getenv("HOME"), ".hearthrc")

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

func is_yes(input string) bool {
	strings.ToLower(input)
	return input[0] == 'y'
}

func CreateInteractive(config_path string) (Config, error) {
	var answer string
	var repo_dir string
	var config Config

	default_repo_path := path.Join(os.Getenv("HOME"), ".hearth")

	// entry. should we make a new file?
	fmt.Printf("no %s found, create a new one? [y/n] ", default_repo_path)
	if fmt.Scanln(&answer); is_yes(answer) == false {
		return config, errors.New("ok then. have a good day!")
	}

	// where should the repo go?
	fmt.Printf("what directory do you wish to use as your repository? [%s] ", default_repo_path)
	if fmt.Scanln(&repo_dir); len(repo_dir) == 0 {
		repo_dir = default_repo_path // use default
	}

	// does that directory exist yet? if so, should we ignore that fact?
	_, err := os.Stat(repo_dir)
	if err == nil {
		fmt.Print("directory exists... create with dirty state? [y/n] ")
		if fmt.Scanln(&answer); is_yes(answer) == false {
			return config, errors.New("will not overwrite existing. aborting.")
		}
	} else {
		err = os.MkdirAll(repo_dir, 0755)
		if err != nil {
			return config, err
		}
	}

	// set our default state
	return Create(os.Getenv("HOME"), ".hearthrc", repo_dir)
}

func Create(conf_path, conf_fname, repo_path string) (Config, error) {
	var config Config
	config.BaseDirectory = repo_path

	// write the config file out
	return config, Write(path.Join(conf_path, conf_fname), config)
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
