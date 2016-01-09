package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	rc "github.com/zmarcantel/hearth/common"

	yaml "gopkg.in/yaml.v2"
)

// Opens the ~/.hearthrc file.
// If this file does not exist, an interactive prompt begins that will attempt to create
// one. Because this function is integral to startup, we pass back a return code.
func Open() (HearthConfig, rc.ReturnCode, error) {
	var config HearthConfig
	default_file := path.Join(os.Getenv("HOME"), ".hearthrc")

	// try to read the default
	config_bytes, err := ioutil.ReadFile(default_file)
	if err != nil {
		// maybe this is a first boot?
		code, err := handle_open_config_error(err)
		if err != nil {
			// clearly not, or something else went wrong
			return config, code, err
		}

		// we must have made a new file, try loading it again
		config_bytes, err = ioutil.ReadFile(default_file)
		if err != nil {
			return config, rc.ReturnReadConfigFileFailure, fmt.Errorf("could not find/open .hearthrc.... %s", err.Error())
		}
	}

	// make a config out of it
	if err = yaml.Unmarshal(config_bytes, &config); err != nil {
		return config, rc.ReturnConfigUnmarshalFailure, fmt.Errorf("failed to parse hearthrc: %s", err.Error())
	}

	return config, rc.ReturnOK, nil
}

func handle_open_config_error(err error) (rc.ReturnCode, error) {
	if os.IsNotExist(err) {
		return CreateInteractive()
	}

	return rc.ReturnNoConfigFile, err
}

func CreateInteractive() (rc.ReturnCode, error) {
	var user_input string
	var user_directory string
	var config HearthConfig
	default_directory := path.Join(os.Getenv("HOME"), ".hearth")

	// entry. should we make a new file?
	fmt.Print("no .hearthrc found, create a new one? [y/n] ")
	fmt.Scanln(&user_input)
	strings.ToLower(user_input)
	if user_input[0] != 'y' {
		return 0, errors.New("ok then. have a good day!")
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
			return 0, errors.New("will not overwrite. aborting.")
		}
	} else {
		err = os.MkdirAll(user_directory, 0755)
		if err != nil {
			return rc.ReturnCreateDirectoryFailure, err
		}
	}

	// set our default state
	config.BaseDirectory = user_directory

	// write the config file out
	return Write(config)
}

func Write(conf HearthConfig) (rc.ReturnCode, error) {
	config_bytes, err := yaml.Marshal(conf)
	if err != nil {
		return rc.ReturnConfigMarshalFailure, fmt.Errorf("could not marshal new config: %s", err.Error())
	}

	err = ioutil.WriteFile(path.Join(os.Getenv("HOME"), ".hearthrc"), config_bytes, 0755)
	if err != nil {
		return rc.ReturnConfigWriteFailure, fmt.Errorf("could not save new config: %s", err.Error())
	}

	return rc.ReturnOK, nil
}

//
// TODO: create non-interactive using CLI
//
