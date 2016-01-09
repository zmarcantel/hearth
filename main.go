package main

import (
	"fmt"
	"log"
	"os"

	"github.com/zmarcantel/hearth/config"

	yaml "gopkg.in/yaml.v2"
)

func main() {
	config, rc, err := config.Open()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(int(rc))
	}

	config_string, err := yaml.Marshal(config)
	if err != nil {
		log.Fatalf("could not circularize the parsing: %v", err) // TODO: obviously this goes away
	}
	fmt.Printf("%s", config_string)
	return
}
