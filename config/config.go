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
func Open() (HearthConfig, error) {
	var config HearthConfig
	default_file := path.Join(os.Getenv("HOME"), ".hearthrc")

	// try to read the default
	config_bytes, err := ioutil.ReadFile(default_file)
	if err != nil {
		return HearthConfig{}, err
	}

	// make a config out of it
	if err = yaml.Unmarshal(config_bytes, &config); err != nil {
		return config, fmt.Errorf("failed to parse hearthrc: %s", err.Error())
	}

	return config, nil
}

func CreateInteractive() (HearthConfig, error) {
	var user_input string
	var user_directory string
	var config HearthConfig
	default_directory := path.Join(os.Getenv("HOME"), ".hearth")

	// entry. should we make a new file?
	fmt.Print("no .hearthrc found, create a new one? [y/n] ")
	fmt.Scanln(&user_input)
	strings.ToLower(user_input)
	if user_input[0] != 'y' {
		return config, errors.New("ok then. have a good day!")
	}

	// where should the repo go?
	fmt.Printf("what directory do you wish to use as your repository? [%s] ", default_directory)
	fmt.Scanln(&user_directory)
	if len(user_directory) == 0 {
		user_directory = default_directory
	}

	// does that directory exist yet? if so, should we ignore that fact?
	_, err := os.Stat(user_directory)
	if err == nil {
		fmt.Print("directory exists... overwrite? [y/n] ")
		fmt.Scanln(&user_input)
		strings.ToLower(user_input)
		if user_input[0] != 'y' {
			return config, errors.New("will not overwrite. aborting.")
		}
	} else {
		err = os.MkdirAll(user_directory, 0755)
		if err != nil {
			return config, err
		}
	}

	// set our default state
	config.BaseDirectory = user_directory

	// write the config file out
	return config, Write(config)
}

func Write(conf HearthConfig) error {
	config_bytes, err := yaml.Marshal(conf)
	if err != nil {
		return fmt.Errorf("could not marshal new config: %s", err.Error())
	}

	err = ioutil.WriteFile(path.Join(os.Getenv("HOME"), ".hearthrc"), config_bytes, 0755)
	if err != nil {
		return fmt.Errorf("could not save new config: %s", err.Error())
	}

	return nil
}

//
// TODO: create non-interactive using CLI
//
