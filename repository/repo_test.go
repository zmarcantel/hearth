package repository

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/libgit2/git2go"
	"github.com/zmarcantel/hearth/config"
)

const default_origin string = "git@github.com:foo/bar.git"

func create_repo(origin string, t *testing.T) Repository {
	rand.Seed(time.Now().UnixNano())
	repo_path := path.Join(os.TempDir(), strconv.FormatUint(uint64(rand.Int63()), 10))

	repo, err := Create(repo_path, origin)
	if err != nil {
		t.Fatalf("could not create repo for test: %s", err.Error())
	}

	if len(repo.Path) == 0 {
		t.Fatalf("repo path name is empty")
	} else if repo.Path != repo_path {
		t.Fatalf("repo path does not match the temp directory we gave it")
	}

	return repo
}

func temp_dir() string {
	rand.Seed(time.Now().UnixNano())
	return path.Join(os.TempDir(), strconv.FormatUint(uint64(rand.Int63()), 10))
}

func create_origin_repo(t *testing.T) (Repository, string) {
	repo_path := temp_dir()

	r, err := git.InitRepository(repo_path, true)
	if err != nil {
		t.Fatal(err)
	}

	return Repository{r, repo_path, config.Config{}}, repo_path
}

func make_filled_dir(repo Repository, files int, t *testing.T) {
	dir, err := ioutil.TempDir(repo.Path, "")
	if err != nil {
		t.Fatal(err)
	}

	// make some new files
	for f := 0; f < files; f += 1 {
		file, err := ioutil.TempFile(dir, "")
		if err != nil {
			t.Fatalf("could not create tempfile: %s", err.Error())
		}
		file.Close()
	}
}

func TestCreate_DirectoryCreated(t *testing.T) {
	repo := create_repo(default_origin, t)
	defer os.RemoveAll(repo.Path)

	if _, err := os.Stat(repo.Path); err != nil {
		t.Errorf("did not find the directory the repo should have created: %s", repo.Path)
	}
}

func TestCreate_ConfigCreated(t *testing.T) {
	repo := create_repo(default_origin, t)
	defer os.RemoveAll(repo.Path)

	if _, err := os.Stat(path.Join(repo.Path, config.Name)); err != nil {
		t.Errorf("did not find the config file the repo should have created: %s", repo.Path)
	}
}

func TestCreate_ConfigHasRepoDirectory(t *testing.T) {
	repo := create_repo(default_origin, t)
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
	repo := create_repo(default_origin, t)
	defer os.RemoveAll(repo.Path)

	lst, err := repo.Remotes.List()
	if err != nil {
		t.Error(err)
	}

	if len(lst) != 1 {
		t.Errorf("repo has %d remotes, but we expected 1 (origin) to exist", len(lst))
	}

	if _, err := repo.Remotes.Lookup("origin"); err != nil {
		t.Errorf("could not find remote:origin in repo")
	}
}

func TestCommitThenPush(t *testing.T) {
	origin, origin_path := create_origin_repo(t)
	repo := create_repo(origin_path, t)

	defer os.RemoveAll(origin_path)
	defer os.RemoveAll(repo.Path)

	num_files := 5
	num_commits := 10

	for i := 0; i < num_commits; i += 1 {
		make_filled_dir(repo, num_files, t)

		// commit and push
		commit_message := fmt.Sprintf("commit #%d", i)
		c, err := repo.CommitAndPush(commit_message, "master")
		if err != nil {
			t.Fatal(err)
		}
		defer c.Free()

		// check that origin has i+1 commits
		expect := i + 1
		count, err := origin.CommitCount()
		if err != nil {
			t.Fatalf("could not count commits: %s", err.Error())
		}
		if count != uint64(expect) {
			t.Fatalf("expected %d commits after commit #%d, but found %d", expect, i, count)
		}
	}
}

func TestPull(t *testing.T) {
	origin, origin_path := create_origin_repo(t)
	repo := create_repo(origin_path, t)
	test_repo_path := temp_dir()

	defer os.RemoveAll(origin_path)
	defer os.RemoveAll(repo.Path)
	defer os.RemoveAll(test_repo_path)
	defer origin.Free()
	defer repo.Free()

	num_files := 5
	num_commits := 10

	var test_repo Repository
	for i := 0; i < num_commits; i += 1 {
		make_filled_dir(repo, num_files, t)

		// commit and push
		commit_message := fmt.Sprintf("commit #%d", i)
		c, err := repo.CommitAndPush(commit_message, "master")
		if err != nil {
			t.Fatal(err)
		}
		defer c.Free()

		// check that origin has i+1 commits
		expect := i + 1
		count, err := origin.CommitCount()
		if err != nil {
			t.Fatalf("could not count commits: %s", err.Error())
		}
		if count != uint64(expect) {
			t.Fatalf("expected %d commits after commit #%d, but found %d", expect, i, count)
		}

		if i == num_commits/2 {
			test_repo, err = Clone(test_repo_path, origin_path)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	defer test_repo.Free()

	// check we have comm/2 commites
	count, err := test_repo.CommitCount()
	if err != nil {
		t.Fatal(err)
	}
	if count != uint64(num_commits/2)+1 {
		t.Fatalf("expected test repo to have %d commits but has %d", (num_commits/2)+1, count)
	}

	// pull
	err = test_repo.Pull()
	if err != nil {
		t.Error(err)
	}

	// check we have num_commit commits
	count, err = test_repo.CommitCount()
	if err != nil {
		t.Fatal(err)
	}
	if count != uint64(num_commits)+1 {
		t.Fatalf("expected test repo to have %d commits but has %d", num_commits+1, count)
	}

	files, err := ioutil.ReadDir(test_repo.Path)
	if err != nil {
		t.Fatal(err)
	}

	dirs := 0
	for _, f := range files {
		if f.IsDir() {
			dirs += 1
		}
	}
	if dirs != num_commits {
		t.Fatalf("expected test repo to have %d directories but has %d", num_commits, dirs)
	}
}
