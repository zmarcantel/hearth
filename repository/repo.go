package repository

import (
	"errors"
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

func Clone(path, repository string) (Repository, error) {
	repo_raw, err := git.Clone(repository, path, &git.CloneOptions{})
	if err != nil {
		return Repository{}, err
	}

	repo := Repository{repo_raw, path, config.Config{}}
	if err := repo.InitFiles(); err != nil {
		return repo, err
	}

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

	if err := repo.InitFiles(); err != nil {
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
func (r *Repository) InitFiles() error {
	config_path := path.Join(r.Path, config.Name) // we create the config inside the repo
	r.Config.BaseDirectory = r.Path               // save the path in the config

	return r.Config.Write(config_path)
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
	defer origin.Free()

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

func credentialsCallback(url, username string, allowedTypes git.CredType) (git.ErrorCode, *git.Cred) {
	ret, cred := git.NewCredSshKeyFromAgent(username)
	code := git.ErrorCode(ret)

	// TODO: get user:pass in cli.... add as args?

	return code, &cred // TODO: return ptr? seems weird
}

func certificateCheckCallback(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
	// TODO: !!!!!!!!!!!!!!!!
	return git.ErrorCode(0)
}

func completionCallback(remote git.RemoteCompletion) git.ErrorCode {
	fmt.Println(remote)
	return git.ErrOk
}

func (r Repository) Pull() error {
	origin, err := r.Remotes.Lookup("origin")
	if err != nil {
		return err
	}
	defer origin.Free()

	fetch_opts := git.FetchOptions{
		Prune:           git.FetchPruneUnspecified,
		DownloadTags:    git.DownloadTagsAll,
		UpdateFetchhead: true,
		RemoteCallbacks: git.RemoteCallbacks{
			CredentialsCallback:      credentialsCallback,
			CertificateCheckCallback: certificateCheckCallback,
			CompletionCallback:       completionCallback,
		},
	}

	err = origin.Fetch([]string{"refs/heads/master"}, &fetch_opts, "") // TODO: do not assume master
	if err != nil {
		return err
	}

	remoteBranch, err := r.References.Lookup("refs/remotes/origin/master") // TODO: do not assume master
	if err != nil {
		return err
	}

	head, err := r.Head()
	if err != nil {
		return err
	}

	remoteBranchID := remoteBranch.Target()
	annotatedCommit, err := r.AnnotatedCommitFromRef(remoteBranch)
	if err != nil {
		return err
	}

	// Do the merge analysis
	mergeHeads := make([]*git.AnnotatedCommit, 1)
	mergeHeads[0] = annotatedCommit
	analysis, _, err := r.MergeAnalysis(mergeHeads)
	if err != nil {
		return err
	}

	// nothing to do
	if analysis&git.MergeAnalysisUpToDate != 0 {
		fmt.Println("ALready up to date.")
		return nil
	} else if analysis&git.MergeAnalysisNormal != 0 {
		// Just merge changes
		if err := r.Merge([]*git.AnnotatedCommit{annotatedCommit}, nil, nil); err != nil {
			return err
		}
		// Check for conflicts
		index, err := r.Index()
		if err != nil {
			return err
		}

		if index.HasConflicts() {
			// TODO: list the conflicting files
			return errors.New("Conflicts encountered. Please resolve them.")
		}

		// Make the merge commit
		sig, err := r.DefaultSignature()
		if err != nil {
			return err
		}

		// Get Write Tree
		treeId, err := index.WriteTree()
		if err != nil {
			return err
		}

		tree, err := r.LookupTree(treeId)
		if err != nil {
			return err
		}

		localCommit, err := r.LookupCommit(head.Target())
		if err != nil {
			return err
		}

		remoteCommit, err := r.LookupCommit(remoteBranchID)
		if err != nil {
			return err
		}

		r.CreateCommit("HEAD", sig, sig, "", tree, localCommit, remoteCommit)

		// Clean up
		r.StateCleanup()
	} else if analysis&git.MergeAnalysisFastForward != 0 {
		// Fast-forward changes
		// Get remote tree
		remoteTree, err := r.LookupTree(remoteBranchID)
		if err != nil {
			return err
		}

		// Checkout
		if err := r.CheckoutTree(remoteTree, nil); err != nil {
			return err
		}

		branchRef, err := r.References.Lookup("refs/heads/master") // TODO: not just master
		if err != nil {
			return err
		}

		// Point branch to the object
		branchRef.SetTarget(remoteBranchID, "")
		if _, err := head.SetTarget(remoteBranchID, ""); err != nil {
			return err
		}

	}

	return nil
}

// Holds metadata and creates an action point for packages.
type PackageInfo struct {
	Name          string
	InstalledPath string
}
