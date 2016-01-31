package config

import (
	"testing"

	"gopkg.in/yaml.v2"
)

//==================================================
// Install Packages
//==================================================

func TestInstallPackage_Unmarshal_String(t *testing.T) {
	test := `this is my command`

	var conf InstallPackage
	err := yaml.Unmarshal([]byte(test), &conf)
	if err != nil {
		t.Error(err)
	}

	if conf.Cmd != "this is my command" {
		t.Errorf("wrong command: '%s'", conf.Cmd)
	}

	if conf.PreCmd != "" {
		t.Error("should not have a pre-command")
	}

	if conf.PostCmd != "" {
		t.Error("should not have a post-command")
	}
}

func TestInstallPackage_Unmarshal_List(t *testing.T) {
	test := `
pre: erp
cmd: dmc
post: tsop
`

	var conf InstallPackage
	err := yaml.Unmarshal([]byte(test), &conf)
	if err != nil {
		t.Error(err)
	}

	if conf.Cmd != "dmc" {
		t.Errorf("wrong command: '%s'", conf.Cmd)
	}

	if conf.PreCmd != "erp" {
		t.Errorf("wrong pre-command: '%s'", conf.PreCmd)
	}

	if conf.PostCmd != "tsop" {
		t.Errorf("wrong post-command: '%s'", conf.PostCmd)
	}
}

//==================================================
// Package Map
//==================================================

func TestPackageMap_Unmarshal_NameInserted(t *testing.T) {
	test := `
first:
    stow_args: empty

second:
    stow_args: empty

third:
    stow_args: empty
`

	var conf PackageMap
	err := yaml.Unmarshal([]byte(test), &conf)
	if err != nil {
		t.Error(err)
	}

	if len(conf) != 3 {
		t.Errorf("expected 3 entries, found %d", len(conf))
	}

	for _, name := range []string{"first", "second", "third"} {
		c, exists := conf[name]
		if !exists {
			t.Errorf("expected '%s' to be in map", name)
		} else {
			if c.Name != name {
				t.Errorf("expected name='%s', but got '%s'", name, c.Name)
			}
		}
	}
}
