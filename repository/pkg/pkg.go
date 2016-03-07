package pkg

import (
	"bytes"
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
	Stow    bool   `yaml:",omitempty"`
}

func (i Install) RunAll(wd string) error {
	if len(i.Cmd) == 0 {
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
	cmd_str = os.ExpandEnv(cmd_str)
	cmd_raw := strings.Split(cmd_str, " ")
	if length := len(cmd_raw); length == 0 {
		// TODO: hmmm can this happen?
		return nil
	} else if length == 1 {
		cmd_raw = append(cmd_raw, "")
	}

	// TODO: add env vars

	var out bytes.Buffer
	cmd := exec.Command(cmd_raw[0], cmd_raw[1:]...)
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		fmt.Println(out.String())
		return err
	}

	return nil
}

func (i Install) MarshalYAML() (interface{}, error) {
	if len(i.PreCmd) > 0 || len(i.PostCmd) > 0 {
		return i, nil
	}

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
	pushd, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(pushd)

	if len(u.Once) != 0 {
		if err := u.run(u.Once, root, ""); err != nil {
			return err
		}
	}

	return filepath.Walk(root, func(p string, i os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// directory command
		if i.IsDir() && len(u.Directory) != 0 {
			if err := u.run(u.Directory, p, ""); err != nil {
				return fmt.Errorf("could not run directory comand: %s", err.Error())
			}
		}

		// file command
		if i.IsDir() == false && len(u.File) != 0 {
			if err != nil {
				return fmt.Errorf("could not set hearth file env: %s", err.Error())
			}

			if err := u.run(u.File, filepath.Dir(p), p); err != nil {
				return fmt.Errorf("could not run file comand: %s", err.Error())
			}
		}

		return nil
	})

}

func (u Update) run(cmd_str, wd, fname string) error {
	// set env
	if len(wd) > 0 {
		if err := os.Setenv("HEARTH_DIR", wd); err != nil {
			return err
		}
	}
	if len(fname) > 0 {
		if err := os.Setenv("HEARTH_FILE", fname); err != nil {
			return err
		}
	}

	// expand.... exec has weird issues with expanding cmd.Env
	cmd_str = os.ExpandEnv(cmd_str)
	cmd_str = strings.TrimSpace(cmd_str)

	cmd_array := []string{"-c", cmd_str}

	// keep an output buffer
	var out bytes.Buffer
	cmd := exec.Command("/bin/bash", cmd_array...) // TODO: certainly not ideal
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Env = os.Environ()
	cmd.Dir = wd

	// ... and go
	err := cmd.Run()
	if err != nil && u.IgnoreErrors == false {
		fmt.Println(out.String())
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
	UpdateCmd  Update  `yaml:"update,omitempty"`
	InstallCmd Install `yaml:"install,omitempty"` // mutually exclusive with Target
	Target     string  `yaml:",omitempty"`        // mutually exclusive with Install
}

func (i Info) Install(wd string) error {
	if strings.HasPrefix(i.Target, "~/") {
		i.Target = path.Join(os.Getenv("HOME"), i.Target[2:])
	}

	all_files := false
	target := i.Target
	if strings.HasPrefix(i.Target, "all:") {
		all_files = true
		target = target[4:]
	}
	if strings.HasPrefix(target, "~/") {
		target = path.Join(os.Getenv("HOME"), target[2:])
	}

	fmt.Println(i.Target)

	// if we have a target, then symlink and shortcircuit the rest of the install
	if all_files {
		top_levels, err := filepath.Glob(filepath.Join(wd, "*"))
		if err != nil {
			log.Fatal(err)
		}

		for _, p := range top_levels {
			target = path.Join(target, path.Base(p))

			fmt.Printf("            --> %s\n", target)
			if err := os.Symlink(p, target); err != nil {
				log.Println(err)
			}
		}

		return nil
	} else if len(i.Target) > 0 {
		target = path.Join(target, i.Name)
		if err := os.Symlink(wd, target); err != nil {
			log.Fatal(err)
		}
		return nil
	}

	// if we do not have a regular command, abort
	if len(i.InstallCmd.Cmd) == 0 {
		return nil
	}

	return i.InstallCmd.RunAll(wd)
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
