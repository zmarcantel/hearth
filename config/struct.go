package config

import "github.com/zmarcantel/hearth/repository/pkg"

//==================================================
// Package management
//==================================================

type PackageMap map[string]pkg.Info

func (c *PackageMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = make(PackageMap)

	if err := unmarshal((*map[string]pkg.Info)(c)); err != nil {
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
	BaseDirectory string `yaml:"directory"`
	Packages      PackageMap
}
