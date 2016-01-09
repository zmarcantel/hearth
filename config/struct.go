package config

import (
	"errors"
	"fmt"

	yaml "gopkg.in/yaml.v2"
)

//==================================================
// Update Target Enum
//==================================================

type UpdateTarget int

const (
	Once UpdateTarget = iota
	Directory
	File
)

func (t *UpdateTarget) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var iter string
	unmarshal(&iter)
	switch iter {
	case "once":
		*t = Once
	case "directory":
		*t = Directory
	case "file":
		*t = File
	default:
		return errors.New("unknown update iter type: " + iter)
	}

	return nil
}

func (t UpdateTarget) MarshalYAML() (interface{}, error) {
	switch t {
	case Once:
		return "once", nil
	case Directory:
		return "directory", nil
	case File:
		return "file", nil
	default:
		return nil, fmt.Errorf("unknown update target type: %+v", t)
	}
}

//==================================================
// Installation config
//==================================================

type InstallConfig struct {
	Cmd string `yaml:"shell,omitempty"`
}

func (i InstallConfig) MarshalYAML() (interface{}, error) {
	return i.Cmd, nil
}

func (i *InstallConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	err := unmarshal(&i.Cmd)

	if err != nil {
		return err
	}
	return err
}

//==================================================
// Update config
//==================================================

type UpdateConfig struct {
	Cmd          string       `yaml:"-"`
	Target       UpdateTarget `yaml:"-"`
	IgnoreErrors bool         `yaml:"ignore_errors,omitempty"`
}

func (u UpdateConfig) MarshalYAML() (interface{}, error) {
	return u.Cmd, nil
}

//
// config map

type UpdateConfigMap map[UpdateTarget]UpdateConfig

func (u UpdateConfigMap) MarshalYAML() (interface{}, error) {
	ignore := false
	tmp := make(map[string]string)
	for k, v := range u {
		if !ignore && v.IgnoreErrors {
			ignore = true
		}

		target, err := k.MarshalYAML()
		if err != nil {
			return nil, err
		}

		val, err := v.MarshalYAML()
		if err != nil {
			return nil, err
		}

		tmp[target.(string)] = val.(string)
	}

	if ignore {
		tmp["ignore_errors"] = "true"
	}

	return tmp, nil
}

func (u *UpdateConfigMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	tmp := make(map[string]string)
	if err := unmarshal(&tmp); err != nil {
		return err
	}

	_, ignore_err := tmp["ignore_errors"]
	if ignore_err {
		if err := yaml.Unmarshal([]byte(tmp["ignore_errors"]), &ignore_err); err != nil {
			return err
		}
	}
	delete(tmp, "ignore_errors")

	*u = make(UpdateConfigMap)
	var target UpdateTarget
	for k, v := range tmp {
		if err := yaml.Unmarshal([]byte(k), &target); err != nil {
			return err
		}
		(*u)[target] = UpdateConfig{Cmd: v, Target: target, IgnoreErrors: ignore_err}
	}

	return nil
}

//==================================================
// Environment config
//==================================================

type Environment struct {
	Name    string          `yaml:"-"`
	Install InstallConfig   `yaml:",omitempty"`
	Update  UpdateConfigMap `yaml:",omitempty"`
}

//
// environment map

type EnvironmentMap map[string]Environment

func (e *EnvironmentMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal((*map[string]Environment)(e)); err != nil {
		return err
	}

	for k, v := range *e {
		v.Name = k
		(*e)[k] = v
	}

	return nil
}

//==================================================
// General config management
//==================================================

type Config struct {
	Name     string          `yaml:"-"`
	StowArgs string          `yaml:"stow_args,omitempty"`
	Update   UpdateConfigMap `yaml:"update,omitempty"`
	Install  InstallConfig   `yaml:"install,omitempty"`
}

type ConfigMap map[string]Config

func (c *ConfigMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = make(ConfigMap)

	if err := unmarshal((*map[string]Config)(c)); err != nil {
		return err
	}

	for k, v := range *c {
		v.Name = k
	}

	return nil
}

//==================================================
// Base structure
//==================================================

type HearthConfig struct {
	BaseDirectory string         `yaml:"directory"`
	Environments  EnvironmentMap //`yaml:",inline"`
	Configs       ConfigMap      //`yaml:",omitempty"`
}
