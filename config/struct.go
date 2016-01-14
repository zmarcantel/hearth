package config

//==================================================
// Installation config
//==================================================

type InstallConfig struct {
	PreCmd  string `yaml:"pre,omitempty"`
	Cmd     string `yaml:"cmd,omitempty"`
	PostCmd string `yaml:"post,omitempty"`
}

func (i InstallConfig) MarshalYAML() (interface{}, error) {
	return i.Cmd, nil
}

type installConfigUnmarshaler InstallConfig

func (i *InstallConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	as_str_err := unmarshal(&i.Cmd)
	if as_str_err == nil {
		return nil
	}

	// do like normal but avoid recursion
	return unmarshal((*installConfigUnmarshaler)(i))
}

//==================================================
// Update config
//==================================================

type UpdateConfig struct {
	Once         string `yaml:"once,omitempty"`
	File         string `yaml:"file,omitempty"`
	Directory    string `yaml:"directory,omitempty"`
	IgnoreErrors bool   `yaml:"ignore_errors,omitempty"`
}

//
// update config map
type UpdateConfigMap map[string]UpdateConfig

// TODO: string-or-list like the install config

//==================================================
// Environment config
//==================================================

type Environment struct {
	Name    string        `yaml:"-"`
	Install InstallConfig `yaml:",omitempty"`
	Update  UpdateConfig  `yaml:",omitempty"`
}

//
// environment map

type EnvironmentMap map[string]Environment

func (e *EnvironmentMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// fill in e using a cast
	if err := unmarshal((*map[string]Environment)(e)); err != nil {
		return err
	}

	// iterate ourself and fill key->val.Name
	for k, v := range *e {
		v.Name = k
		(*e)[k] = v
	}

	return nil
}

//==================================================
// General config management
//==================================================

type ConfigSection struct {
	Name     string        `yaml:"-"`
	StowArgs string        `yaml:"stow_args,omitempty"`
	Update   UpdateConfig  `yaml:"update,omitempty"`
	Install  InstallConfig `yaml:"install,omitempty"`
}

type ConfigMap map[string]ConfigSection

func (c *ConfigMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = make(ConfigMap)

	if err := unmarshal((*map[string]ConfigSection)(c)); err != nil {
		return err
	}

	for k, v := range *c {
		v.Name = k
		(*c)[k] = v // TODO: probably a more efficient way than updating entire key
	}

	return nil
}

//==================================================
// Base structure
//==================================================

type Config struct {
	BaseDirectory string         `yaml:"directory"`
	Environments  EnvironmentMap //`yaml:",inline"`
	Configs       ConfigMap      //`yaml:",omitempty"`
}
