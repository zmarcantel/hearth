package repository

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/zmarcantel/hearth/config"

	git "github.com/libgit2/git2go"
)

// Convenience method for getting the default path for a repository (~/.hearth)
func DefaultPath() string {
	return path.Join(os.Getenv("HOME"), ".hearth")
}

// Holds all the metadata and the actual repository information. Central "actor"
// in the system as almost all commands are derived from manipulating this struct.
type Repository struct {
	*git.Repository
	Path   string
	Config config.Config
}

// Open the managed repository. Opens the config, gets the repo directory, and
// fills all the necessary data
func Open() (Repository, error) {
	var repo Repository
	conf, err := config.Open()
	if err != nil {
		return repo, err
	}

	repo_raw, err := git.OpenRepository(conf.BaseDirectory)
	if err != nil {
		return repo, fmt.Errorf("could not open git repository: %s", err)
	}
	repo = Repository{repo_raw, conf.BaseDirectory, conf}

	return repo, nil
}

// Create a new repository in the given path and fill it with a remote:origin if given.
// This also creates the hearth config (and all other InitFiles()) in the repo.
func Create(path, origin string) (Repository, error) {
	if _, err := os.Stat(path); err == nil {
		return Repository{}, fmt.Errorf("%s already exists.", path)
	}

	repo_raw, err := git.InitRepository(path, false)
	if err != nil {
		return Repository{}, err
	}
	repo := Repository{repo_raw, path, config.Config{}}

	if err := InitFiles(&repo); err != nil {
		return repo, err
	}

	if len(origin) == 0 {
		fmt.Println("WARN: origin not provided. must be added before issuing a save command")
	} else {
		_, err := repo.Remotes.Create("origin", origin)
		if err != nil {
			log.Fatalf("could not add remote:origin to repo: %s", err.Error())
		}
	}

	return repo, nil
}

// Creates the needed (or later, wanted) files in a repository. Primarily, this is
// used for generating a config on creation of a new repo
func InitFiles(repo *Repository) error {
	config_path := path.Join(repo.Path, config.Name) // we create the config inside the repo
	repo.Config.BaseDirectory = repo.Path            // save the path in the config

	return repo.Config.Write(config_path)
}

// Truthy function on whether the repository has the given package or not.
// Case sensitivity is that of the underlying filesystem.
func (r Repository) HasPackage(name string) bool {
	s, err := os.Stat(path.Join(r.Path, name))
	return err == nil && s.IsDir()
}

// Get the package information for the package with the given name.
// Boolean in return is existence check.
func (r Repository) GetPackage(name string) (PackageInfo, bool) {
	src_path := path.Join(r.Path, name)
	s, err := os.Stat(src_path)
	if err != nil {
		return PackageInfo{}, false
	}

	result := PackageInfo{Name: name, InstalledPath: src_path}
	return result, s.IsDir()
}

// Essentially `git add --all .` in thre repo directory, and commit with the given message.
// TODO: use save command deltas to auto-gen commit message
func (r Repository) CommitAll(message string) (*git.Commit, error) {
	if len(message) == 0 {
		return nil, fmt.Errorf("a commit message is currently required")
	}

	// get the commiter info
	sig, err := r.DefaultSignature()
	if err != nil {
		return nil, fmt.Errorf("could not get signature for commit: %s", err.Error())
	}

	// get the HEAD index
	idx, err := r.Index()
	if err != nil {
		return nil, fmt.Errorf("could not get repo index: %s", err.Error())
	}

	// basically, like running `git add --all .` in the repo directory
	err = idx.AddAll([]string{r.Path}, git.IndexAddDefault, nil)
	if err != nil {
		return nil, fmt.Errorf("could not add files to commit: %s", err.Error())
	}

	// finalize the diff tree
	tree_id, err := idx.WriteTree()
	if err != nil {
		return nil, fmt.Errorf("could not write tree to repo: %s", err.Error())
	}

	// get the tree
	tree, err := r.LookupTree(tree_id)
	if err != nil {
		return nil, fmt.Errorf("could not get commit's tree: %s", err.Error())
	}
	defer tree.Free()

	// get the HEAD index
	var commit_id *git.Oid
	head, err := r.Head()
	if err != nil { // first commit....
		// use the above data to finalize the commit
		commit_id, err = r.CreateCommit("HEAD", sig, sig, message, tree)
		if err != nil {
			return nil, fmt.Errorf("could not create commit: %s", err.Error())
		}
	} else { // .. have a tip
		tip, err := r.LookupCommit(head.Target())
		if err != nil {
			return nil, fmt.Errorf("could not get repo HEAD commit: %s", err.Error())
		}
		defer tip.Free()

		// use the above data to finalize the commit
		commit_id, err = r.CreateCommit("HEAD", sig, sig, message, tree, tip)
		if err != nil {
			return nil, fmt.Errorf("could not create commit: %s", err.Error())
		}
	}

	commit, err := r.LookupCommit(commit_id)
	if err != nil {
		return nil, fmt.Errorf("could not lookup the commit: %s", err.Error())
	}

	return commit, nil
}

// TODO: docs
func (r Repository) Push(branch string) error {
	// make sure we have an origin to push to
	origin, err := r.Remotes.Lookup("origin")
	if err != nil {
		log.Fatal("remote:origin does not exist in repository")
	}

	// TODO: sanitize the branch
	branch = path.Join("refs/heads/", branch)
	return origin.Push([]string{branch}, nil)
}

// Commit all changes in the repo with the given message. Subsequently,
// push this commit to the given branch on origin.
func (r Repository) CommitAndPush(message, branch string) (*git.Commit, error) {
	if len(branch) == 0 {
		branch = "master"
	}

	commit, err := r.CommitAll(message)
	if err != nil {
		return nil, err
	}

	return commit, r.Push(branch)
}

// Iterate the branch by TOPO|TIME and count the preceding commits from HEAD
func (r Repository) CommitCount() (uint64, error) {
	// get a revwalk
	walk, err := r.Walk()
	if err != nil {
		return 0, fmt.Errorf("could not start walk: %s", err.Error())
	}

	walk.Sorting(git.SortTopological | git.SortTime)
	err = walk.PushHead()
	if err != nil {
		return 0, fmt.Errorf("could not push head to walk: %s", err.Error())
	}

	var count uint64 = 0
	err = walk.Iterate(func(commit *git.Commit) bool {
		count += 1
		return true
	})

	return count, nil
}

// Holds metadata and creates an action point for packages.
type PackageInfo struct {
	Name          string
	InstalledPath string
}
