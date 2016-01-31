package config

//==================================================
// Installation config
//==================================================

type InstallPackage struct {
	PreCmd  string `yaml:"pre,omitempty"`
	Cmd     string `yaml:"cmd,omitempty"`
	PostCmd string `yaml:"post,omitempty"`
}

func (i InstallPackage) MarshalYAML() (interface{}, error) {
	return i.Cmd, nil
}

type installPackageUnmarshaler InstallPackage

func (i *InstallPackage) UnmarshalYAML(unmarshal func(interface{}) error) error {
	as_str_err := unmarshal(&i.Cmd)
	if as_str_err == nil {
		return nil
	}

	// do like normal but avoid recursion
	return unmarshal((*installPackageUnmarshaler)(i))
}

//==================================================
// Update config
//==================================================

type UpdatePackage struct {
	Once         string `yaml:"once,omitempty"`
	File         string `yaml:"file,omitempty"`
	Directory    string `yaml:"directory,omitempty"`
	IgnoreErrors bool   `yaml:"ignore_errors,omitempty"`
}

//
// update config map
type UpdatePackageMap map[string]UpdatePackage

// TODO: string-or-list like the install config

//==================================================
// Package management
//==================================================

type PackageSection struct {
	Name    string         `yaml:"-"`
	Update  UpdatePackage  `yaml:",omitempty"`
	Install InstallPackage `yaml:",omitempty"` // mutually exclusive with Target
	Target  string         `yaml:",omitempty"` // mutually exclusive with Install
}

type PackageMap map[string]PackageSection

func (c *PackageMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = make(PackageMap)

	if err := unmarshal((*map[string]PackageSection)(c)); err != nil {
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

type Package struct {
	BaseDirectory string `yaml:"directory"`
	Packages      PackageMap
}
