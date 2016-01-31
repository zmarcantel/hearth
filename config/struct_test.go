package config

import (
	"testing"

	"gopkg.in/yaml.v2"
)

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
