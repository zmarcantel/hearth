package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"gopkg.in/yaml.v2"

	"github.com/zmarcantel/hearth/repo"

	"github.com/codegangsta/cli"
)

func main() {
	app := init_flags()
	if err := app.Run(os.Args); err != nil {
		log.Fatalf(err.Error())
	}
}

func start_cli(c *cli.Context) {
	var err error
	var r repo.Repository

	default_file := path.Join(os.Getenv("HOME"), ".hearthrc")

	// check if we have a config file
	if _, err = os.Stat(default_file); os.IsNotExist(err) {
		// config does not exist
		r, err = repo.CreateWithConfig(default_file, true)
		if err != nil {
			log.Fatalf(err.Error())
		}
	} else {
		// config exists
		r, err = repo.Open()
		if err != nil {
			log.Fatalf(err.Error())
		}
	}

	config_string, err := yaml.Marshal(r.Config)
	if err != nil {
		log.Fatalf("could not circularize the parsing: %v", err) // TODO: obviously this goes away
	}

	fmt.Printf("%s", config_string)
}
