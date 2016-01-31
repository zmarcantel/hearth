package pkg

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v2"
)

//==================================================
// Unmarshaling
//==================================================

func TestInstallPackage_Unmarshal_String(t *testing.T) {
	test := `this is my command`

	var conf Install
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

func TestInstall_Unmarshal_List(t *testing.T) {
	test := `
pre: erp
cmd: dmc
post: tsop
`

	var conf Install
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
// helpers
//==================================================

func temp_dir() string {
	rand.Seed(time.Now().UnixNano())
	return path.Join(os.TempDir(), strconv.FormatUint(uint64(rand.Int63()), 10))
}

func mktemp(t *testing.T) string {
	dir := temp_dir()
	if err := os.Mkdir(dir, 0777); err != nil {
		t.Fatal(err)
	}

	return dir
}

func make_dir(prefix string, t *testing.T) string {
	rand.Seed(time.Now().UnixNano())
	dir := path.Join(prefix, strconv.FormatUint(uint64(rand.Int63()), 10))

	err := os.Mkdir(dir, 0777)
	if err != nil {
		t.Fatal(err)
	}

	return dir
}

func expect_file(p, err_str string, t *testing.T) {
	if _, err := os.Stat(p); err != nil {
		t.Errorf(err_str)
	}
}

func expect_no_file(p, err_str string, t *testing.T) {
	if _, err := os.Stat(p); err == nil {
		t.Errorf(err_str)
	}
}

//==================================================
// install tests
//==================================================

func TestInstall_RunAll_NoCmd(t *testing.T) {
	dir := mktemp(t)
	defer os.RemoveAll(dir)

	install := Install{
		PreCmd:  "touch pre.txt",
		PostCmd: "touch post.txt",
	}
	err := install.RunAll(dir)
	if err != nil {
		t.Fatal(err)
	}

	expect_no_file(path.Join(dir, "pre.txt"), "ran pre command", t)
	expect_no_file(path.Join(dir, "post.txt"), "ran post command", t)
}

func TestInstall_RunAll_CmdOnly(t *testing.T) {
	dir := mktemp(t)
	defer os.RemoveAll(dir)

	install := Install{
		Cmd: "touch cmd.txt",
	}
	err := install.RunAll(dir)
	if err != nil {
		t.Fatal(err)
	}

	expect_file(path.Join(dir, "cmd.txt"), "did not run command", t)
}

func TestInstall_RunAll(t *testing.T) {
	dir := mktemp(t)
	defer os.RemoveAll(dir)

	install := Install{
		PreCmd:  "touch pre.txt",
		Cmd:     "touch cmd.txt",
		PostCmd: "touch post.txt",
	}
	err := install.RunAll(dir)
	if err != nil {
		t.Fatal(err)
	}

	expect_file(path.Join(dir, "pre.txt"), "did not run pre command", t)
	expect_file(path.Join(dir, "cmd.txt"), "did not run command", t)
	expect_file(path.Join(dir, "post.txt"), "did not run post command", t)
}

//==================================================
// update tests
//==================================================

func TestUpdate_RunAll_NoCmd(t *testing.T) {
	dir := mktemp(t)
	defer os.RemoveAll(dir)

	sub := make_dir(dir, t)
	if err := ioutil.WriteFile(path.Join(sub, "script.sh"), []byte("#!/bin/bash"), 0644); err != nil {
		t.Fatal(err)
	}

	// make and run the update
	update := Update{
		Once:      "touch test.txt",
		File:      "chmod +x $HEARTH_FILE",
		Directory: "echo $HEARTH_DIR > dir.txt",
	}
	err := update.RunAll(dir)
	if err != nil {
		t.Fatal(err)
	}

	// check expectations
	expect_file(path.Join(dir, "test.txt"), "did not run once command (or in wrong directory)", t)
	expect_file(path.Join(sub, "dir.txt"), "did not run directory command", t)
	if b, err := ioutil.ReadFile(path.Join(sub, "dir.txt")); err != nil {
		t.Error(err)
	} else if strings.TrimSpace(string(b)) != sub {
		t.Errorf("incorrect directory in file. expected '%s' but got '%s'", sub, string(b))
	}

	stat, err := os.Stat(path.Join(sub, "script.sh"))
	if err != nil {
		t.Error(err)
	}

	if stat.Mode()&0x010101 == 0 {
		t.Fatalf("did not make test-script executable")
	}
}
