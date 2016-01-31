package pkg

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

//==================================================
// Installation config
//==================================================

type Install struct {
	PreCmd  string `yaml:"pre,omitempty"`
	Cmd     string `yaml:"cmd,omitempty"`
	PostCmd string `yaml:"post,omitempty"`
}

func (i Install) RunAll() error {
	if len(i.Cmd) == 0 {
		return nil
	}

	if len(i.PreCmd) > 0 {
		if err := i.run(i.PreCmd); err != nil {
			return err
		}
	}

	if err := i.run(i.Cmd); err != nil {
		return err
	}

	if len(i.PostCmd) > 0 {
		if err := i.run(i.PostCmd); err != nil {
			return err
		}
	}

	return nil
}

func (i Install) run(cmd_str string) error {
	cmd_raw := strings.Split(cmd_str, " ")
	if length := len(cmd_raw); length == 0 {
		// TODO: hmmm can this happen?
		return nil
	} else if length == 1 {
		cmd_raw = append(cmd_raw, "")
	}

	cmd := exec.Command(cmd_raw[0], cmd_raw[1:]...)
	err := cmd.Run()
	if err != nil {
		out, out_err := cmd.Output()
		if out_err != nil {
			return err // just bail
		}

		fmt.Println(string(out))
		return err
	}

	return nil
}

func (i Install) MarshalYAML() (interface{}, error) {
	return i.Cmd, nil
}

type installUnmarshaler Install

func (i *Install) UnmarshalYAML(unmarshal func(interface{}) error) error {
	as_str_err := unmarshal(&i.Cmd)
	if as_str_err == nil {
		return nil
	}

	// do like normal but avoid recursion
	return unmarshal((*installUnmarshaler)(i))
}

//==================================================
// Update config
//==================================================

type Update struct {
	Once         string `yaml:"once,omitempty"`
	File         string `yaml:"file,omitempty"`
	Directory    string `yaml:"directory,omitempty"`
	IgnoreErrors bool   `yaml:"ignore_errors,omitempty"`
}

func (u Update) RunAll(root string) error {
	if len(u.Once) != 0 {
		if err := u.run(u.Once); err != nil {
			return err
		}
	}

	return filepath.Walk(root, func(p string, i os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if i.IsDir() {
			pushd, err := os.Getwd()
			if err != nil {
				return err
			}
			defer os.Chdir(pushd)

			// set the env for the command
			wd := path.Join(root, i.Name())
			err = os.Setenv("HEARTH_DIR", wd)
			if err != nil {
				return err
			}

			if err := os.Chdir(wd); err != nil {
				return err
			}

			if len(u.Directory) != 0 {
				if err := u.run(u.Directory); err != nil {
					return fmt.Errorf("could not run directory comand: %s", err.Error())
				}
			}
		} else if len(u.File) != 0 { // not a directory, and we have a per-file command
			fpath := path.Join(root, i.Name())
			err := os.Setenv("HEARTH_FILE", fpath)
			if err != nil {
				return err
			}

			if err := u.run(u.File); err != nil {
				return fmt.Errorf("could not run file comand: %s", err.Error())
			}
		}

		return nil
	})

}

func (u Update) run(cmd_str string) error {
	cmd_raw := strings.Split(cmd_str, " ")
	if length := len(cmd_raw); length == 0 {
		// TODO: hmmm can this happen?
		return nil
	} else if length == 1 {
		cmd_raw = append(cmd_raw, "")
	}

	cmd := exec.Command(cmd_raw[0], cmd_raw[1:]...)
	err := cmd.Run()
	if err != nil && u.IgnoreErrors == false {
		out, out_err := cmd.Output()
		if out_err != nil {
			return err // just bail
		}

		fmt.Println(string(out))
		return err
	}

	return nil
}

//
// update config map
type UpdateMap map[string]Update

// TODO: string-or-list like the install config

//==================================================
// Base Info struct
//==================================================

// Holds metadata and creates an action point for packages.
type Info struct {
	Name       string  `yaml:"-"`
	UpdateCmd  Update  `yaml:",omitempty"`
	InstallCmd Install `yaml:",omitempty"` // mutually exclusive with Target
	Target     string  `yaml:",omitempty"` // mutually exclusive with Install
}

func (i Info) Install(wd string) error {
	// if we have a target, then symlink and shortcircuit the rest of the install
	if len(i.Target) > 0 {
		if err := os.Symlink(wd, i.Target); err != nil {
			log.Fatal(err)
		}
		return nil
	}

	// if we do not have a regular command, abort
	if len(i.InstallCmd.Cmd) == 0 {
		return nil
	}

	// save current dir and defer popping
	pushd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	defer os.Chdir(pushd)

	// move into the package directory
	if err := os.Chdir(wd); err != nil {
		return err
	}

	return i.InstallCmd.RunAll()
}

func (i Info) Update(wd string) error {
	// save current dir and defer popping
	pushd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	defer os.Chdir(pushd)

	// move into the package directory
	if err := os.Chdir(wd); err != nil {
		return err
	}

	return i.UpdateCmd.RunAll(wd)
}
