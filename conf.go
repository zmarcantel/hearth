package main

import (
    "errors"
)

type Environment struct {
    Filename        string      `yaml:"file,omitempty"`
    Arguments       []string    `yaml:"args,omitempty"`
}

type UpdateTarget int
const (
    Directory UpdateTarget = 0
    File
)
func (t UpdateTarget) UnmarshalYAML(unmarshal func(interface{}) error) error {
    var iter string
    unmarshal(&iter)
    switch(iter) {
    case "directory": t = Directory
    case "file": t = File
    default: return errors.New("unknown update iter type: "+iter)
    }

    return nil
}

type InstallConfig struct {
    Command         string      `yaml:"shell,omitempty"`
    Script          string      `yaml:"script,omitempty"`
}

type UpdateConfig struct {
    Do              string      `yaml:"do"`
    ForEvery        UpdateTarget `yaml:"for_every"`
}

type Config struct {
    Name            string      `yaml:"-"`
    TargetDir       string      `yaml:"target_dir,omitempty"`
    StowArgs        string      `yaml:"stow_args,omitempty"`
    Update          UpdateConfig    `yaml:"update,omitempty"`
    Install         InstallConfig   `yaml:"install,omitempty"`
}


type HearthConfig struct {
    Environments    map[string]Environment
    Configs         map[string]Config
}
