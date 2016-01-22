package repository

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/zmarcantel/hearth/config"
)

const origin_string string = "git@github.com:foo/bar.git"

func create_repo(t *testing.T) Repository {
	rand.Seed(time.Now().UnixNano())
	repo_path := path.Join(os.TempDir(), strconv.FormatUint(uint64(rand.Int63()), 10))

	repo, err := Create(repo_path, origin_string)
	if err != nil {
		t.Errorf("could not create repo for test: %s", err.Error())
		return Repository{}
	}

	if len(repo.Path) == 0 {
		t.Errorf("repo path name is empty")
	} else if repo.Path != repo_path {
		t.Errorf("repo path does not match the temp directory we gave it")
	}

	return repo
}

func TestCreate_DirectoryCreated(t *testing.T) {
	repo := create_repo(t)
	defer os.RemoveAll(repo.Path)

	if _, err := os.Stat(repo.Path); err != nil {
		t.Errorf("did not find the directory the repo should have created: %s", repo.Path)
	}
}

func TestCreate_ConfigCreated(t *testing.T) {
	repo := create_repo(t)
	defer os.RemoveAll(repo.Path)

	if _, err := os.Stat(path.Join(repo.Path, config.Name)); err != nil {
		t.Errorf("did not find the config file the repo should have created: %s", repo.Path)
	}
}

func TestCreate_ConfigHasRepoDirectory(t *testing.T) {
	repo := create_repo(t)
	defer os.RemoveAll(repo.Path)

	conf_path := path.Join(repo.Path, config.Name)
	_, err := os.Stat(conf_path)
	if err != nil {
		t.Errorf("did not find the config file the repo should have created: %s", repo.Path)
	}

	conf_bytes, err := ioutil.ReadFile(conf_path)
	if err != nil {
		t.Error(err)
	}

	var conf config.Config
	err = yaml.Unmarshal(conf_bytes, &conf)
	if err != nil {
		t.Error(err)
	}
	if conf.BaseDirectory != repo.Path {
		t.Errorf("incorect repo directory in the config: expected=%s, got=%s", repo.Path, conf.BaseDirectory)
	}
}

func TestCreate_OriginAdded(t *testing.T) {
	repo := create_repo(t)
	defer os.RemoveAll(repo.Path)

	lst, err := repo.Repo.Remotes.List()
	if err != nil {
		t.Error(err)
	}

	if len(lst) != 1 {
		t.Errorf("repo has %d remotes, but we expected 1 (origin) to exist", len(lst))
	}

	if _, err := repo.Repo.Remotes.Lookup("origin"); err != nil {
		t.Errorf("could not find remote:origin in repo")
	}
}
