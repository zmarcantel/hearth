package main

import (
    "os"
    "fmt"
    "log"
    "path"
    "io/ioutil"

     yaml "gopkg.in/yaml.v2"
)


func main() {
    config_bytes, err := ioutil.ReadFile(path.Join(os.Getenv("HOME"), ".hearthrc"))
    if err != nil {
        // TODO: check if NOT_EXIST, and offer to create one (interactively?)
        log.Fatalf("failed to open hearthrc: %s", err.Error())
    }

    var config HearthConfig
    if err = yaml.Unmarshal(config_bytes, &config); err != nil {
        log.Fatalf("failed to parse hearthrc: %s", err.Error())
    }

    config_string, err := yaml.Marshal(config)
    if err != nil {
        log.Fatalf("could not circularize the parsing") // TODO: obviously this goes away
    }
    fmt.Printf("%s", config_string)
    return
}
