package repository

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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

func make_dir(prefix string, t *testing.T) string {
	rand.Seed(time.Now().UnixNano())
	path := path.Join(prefix, strconv.FormatUint(uint64(rand.Int63()), 10))

	err := os.Mkdir(path, 0755)
	if err != nil {
		t.Fatal(err)
	}

	return path
}

func make_file(prefix string, t *testing.T) string {
	rand.Seed(time.Now().UnixNano())
	path := path.Join(prefix, strconv.FormatUint(uint64(rand.Int63()), 10))

	err := ioutil.WriteFile(path, []byte(path), 0755)
	if err != nil {
		t.Fatal(err)
	}

	return path
}

func create_origin_repo(t *testing.T) (Repository, string) {
	repo_path := temp_dir()

	r, err := git.InitRepository(repo_path, true)
	if err != nil {
		t.Fatal(err)
	}

	return Repository{r, repo_path, config.Config{}}, repo_path
}

func dir_walk_counts(path string) (dirs uint64, files uint64, err error) {
	err = filepath.Walk(path, func(p string, i os.FileInfo, e error) error {
		if e != nil {
			return e
		}

		if p == path || strings.Contains(p, ".git") {
			return nil
		}

		if i.IsDir() {
			dirs += 1
		} else {
			files += 1
		}

		return nil
	})

	return
}

func make_filled_dir(path string, files int, t *testing.T) string {
	dir, err := ioutil.TempDir(path, "")
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

	return dir
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
		make_filled_dir(repo.Path, num_files, t)

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
	subdir_depth := 2

	var test_repo Repository
	var commit_path_iter string
	for i := 0; i < num_commits; i += 1 {
		commit_path_iter = make_filled_dir(repo.Path, num_files, t)
		for d := 0; d < subdir_depth; d += 1 {
			commit_path_iter = make_filled_dir(commit_path_iter, num_files, t)
		}

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

	// get a count of directories and files in the repo
	dirs, files, err := dir_walk_counts(test_repo.Path)
	if err != nil {
		t.Fatal(err)
	}

	// assert directory count
	expect_dirs := uint64((num_commits * subdir_depth) + num_commits)
	if dirs != expect_dirs {
		t.Fatalf("expected test repo to have %d directories but has %d", expect_dirs, dirs)
	}

	// assert file count
	expect_files := uint64((expect_dirs * uint64(num_files)) + 1) // have to add the .hearthrc
	if files != expect_files {
		t.Fatalf("expected test repo to have %d files but has %d", expect_files, files)
	}
}

func TestChangesFiles(t *testing.T) {
	repo := create_repo(default_origin, t)
	defer os.RemoveAll(repo.Path)
	defer repo.Free()

	// make changes
	num_files := 5
	changed_paths := make([]string, 0)
	base_path := make_dir(repo.Path, t) // first dir
	changed_paths = append(changed_paths, base_path)
	changed_paths = append(changed_paths, make_dir(base_path, t)) // second level

	in_dir := changed_paths[len(changed_paths)-1]
	for i := 0; i < num_files; i += 1 {
		changed_paths = append(changed_paths, make_file(in_dir, t))
	}

	// commit
	c, err := repo.CommitAll("test commit")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Free()

	// check counts
	expect := len(changed_paths) + 1 // .hearthrc
	changed, err := repo.ChangedInLastCommit()
	if err != nil {
		t.Fatal(err)
	}
	changed_count := len(changed)
	if changed_count != expect {
		t.Fatalf("expected %d changed files, got %d", expect, changed_count)
	}

	// we have to sort the array first :/
	sort.StringSlice(changed_paths).Sort()

	// make sure all files in changed are in our created list
	for _, f := range changed {
		actual := path.Join(repo.Path, f)
		if sort.SearchStrings(changed_paths, actual) == len(changed_paths) {
			t.Fatalf("%s not in the list of files we made", actual)
		}
	}
}

func TestIsPackage(t *testing.T) {
	repo := create_repo(default_origin, t)
	defer os.RemoveAll(repo.Path)
	defer repo.Free()

	is_pkg := path.Join(repo.Path, "this_is_a_package")
	not_pkg := path.Join(repo.Path, "file_in/some_dir")
	not_rel := "/var/log/hearth"

	if repo.IsPackage(is_pkg) == false {
		t.Errorf("did not think %s was a package", is_pkg)
	}
	if repo.IsPackage(not_pkg) {
		t.Errorf("thought %s was a package", is_pkg)
	}
	if repo.IsPackage(not_rel) {
		t.Errorf("thought %s was a package", is_pkg)
	}
}

func TestModifiedInLast(t *testing.T) {
	repo := create_repo(default_origin, t)
	defer os.RemoveAll(repo.Path)
	defer repo.Free()

	f := make_file(repo.Path, t)

	c, err := repo.CommitAll("test commit")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Free()

	if repo.ModifiedInLast(filepath.Base(f)) {
		t.Fatalf("modified, not created")
	}

	err = ioutil.WriteFile(f, []byte("some new info"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	second_commit, err := repo.CommitAll("another commit")
	if err != nil {
		t.Fatal(err)
	}
	defer second_commit.Free()

	if repo.ModifiedInLast(filepath.Base(f)) == false {
		t.Fatalf("did not consider it modified")
	}
}

func TestCreatedInLast(t *testing.T) {
	repo := create_repo(default_origin, t)
	defer os.RemoveAll(repo.Path)
	defer repo.Free()

	f := make_file(repo.Path, t)

	c, err := repo.CommitAll("test commit")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Free()

	if repo.CreatedInLast(filepath.Base(f)) == false {
		t.Fatalf("did not consider it created")
	}

	err = ioutil.WriteFile(f, []byte("some new info"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	second_commit, err := repo.CommitAll("another commit")
	if err != nil {
		t.Fatal(err)
	}
	defer second_commit.Free()

	if repo.CreatedInLast(filepath.Base(f)) {
		t.Fatalf("considered created even when it was modified")
	}
}

func TestNewBranch(t *testing.T) {
	repo := create_repo(default_origin, t)
	defer os.RemoveAll(repo.Path)
	defer repo.Free()

	// must have a commit to have a HEAD to have a branch :/
	make_file(repo.Path, t)
	c, err := repo.CommitAll("test commit")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Free()

	// test branching
	name := "test"

	_, err = repo.NewBranch(name)
	if err != nil {
		t.Fatal(err)
	}

	_, err = repo.LookupBranch(name, git.BranchLocal)
	if err != nil {
		t.Fatalf("could not lookup new branch: %s", err.Error())
	}
}

func TestCheckoutBranch(t *testing.T) {
	repo := create_repo(default_origin, t)
	defer os.RemoveAll(repo.Path)
	defer repo.Free()

	// must have a commit to have a HEAD to have a branch :/
	make_file(repo.Path, t)
	c, err := repo.CommitAll("test commit")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Free()

	// test branching
	name := "test"

	branch, err := repo.NewBranch(name)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.CheckoutBranch(branch)
	if err != nil {
		t.Fatalf("could not checkout branch: %s", err.Error())
	}
}

func TestCheckoutBranchByName(t *testing.T) {
	repo := create_repo(default_origin, t)
	defer os.RemoveAll(repo.Path)
	defer repo.Free()

	// must have a commit to have a HEAD to have a branch :/
	make_file(repo.Path, t)
	c, err := repo.CommitAll("test commit")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Free()

	// test branching
	name := "test"

	_, err = repo.NewBranch(name)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.CheckoutBranchByName(name)
	if err != nil {
		t.Fatalf("could not checkout branch: %s", err.Error())
	}
}

// TODO: test checkout branch with dirty working directory
