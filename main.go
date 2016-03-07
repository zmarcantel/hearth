package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/zmarcantel/hearth/config"
	"github.com/zmarcantel/hearth/repository"
	"github.com/zmarcantel/hearth/repository/pkg"

	"github.com/codegangsta/cli"
	git "gopkg.in/libgit2/git2go.v23"
)

func main() {
	app := init_flags()
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

//==================================================
// default action
//==================================================

// TODO: what should this be?
func action_default(ctx *cli.Context) {
	repo, err := repository.Open()
	if err != nil {
		log.Fatal(err)
	}

	index, err := repo.Index()
	if err != nil {
		log.Fatalf("could not get index: %s", err.Error())
	}
	defer index.Free()

	diff_opts := git.DiffOptions{
		Flags:            git.DiffNormal,
		IgnoreSubmodules: git.SubmoduleIgnoreAll,
		Pathspec:         []string{repo.Path},
		ContextLines:     4,
	}

	diff, err := repo.DiffIndexToWorkdir(index, &diff_opts)
	if err != nil {
		log.Fatalf("could not diff index to workdir: %s", err.Error())
	}
	defer diff.Free()

	err = diff.ForEach(func(delta git.DiffDelta, idk float64) (git.DiffForEachHunkCallback, error) {
		// TODO: is the float percentage changed?

		// if rename
		if delta.OldFile.Path != delta.NewFile.Path {
			// TODO: colored output based on delta.Similarity
			fmt.Printf("[ renamed ] %s  -->  %s\n", delta.OldFile.Path, delta.NewFile.Path)
		} else {
			fmt.Printf("[ changed ] %s", delta.NewFile.Path)
		}

		return nil, nil
	}, git.DiffDetailFiles)
	if err != nil {
		log.Fatalf("could not iterate diffs: %s", err.Error())
	}
}

//==================================================
// init action
//==================================================
func action_init(ctx *cli.Context) {
	// get the default repo path (overwritten below) and the forced config path
	repo_path := repository.DefaultPath()
	config_final_path := config.Path()

	// if the user wants a different repo, let them
	if ctx.IsSet("repo") {
		repo_path = opts.RepoPath
	}

	// create the repo (and starter config + any misc files)
	repo, err := repository.Create(repo_path, opts.RepoOrigin)
	if err != nil {
		log.Fatalf("could not initialize repository: %s", err.Error())
	}
	defer repo.Free()

	// symlink {REPO_DIR}/.hearthrc --> $HOME/.hearthrc
	config_src_path := path.Join(repo.Path, config.Name)
	if err := os.Symlink(config_src_path, config_final_path); err != nil {
		log.Fatalf("could not link hearth config into home directory: %s", err.Error())
	}
}

//==================================================
// env action
//==================================================

func action_env(ctx *cli.Context) {
	if len(ctx.Args()) < 1 {
		log.Fatalf("no branch name given")
	} else if len(ctx.Args()) > 1 {
		log.Fatalf("too many branch names given")
	}

	repo, err := repository.Open()
	if err != nil {
		log.Fatal(err)
	}

	// TODO: pull first to get new branches

	branch_name := ctx.Args()[0]
	branch, err := repo.LookupBranch(branch_name, git.BranchLocal)
	if err != nil && ctx.Bool("no-create") == false {
		// no branch so make one
		branch, err = repo.NewBranch(branch_name)
		if err != nil {
			log.Fatal(err)
		}
	}
	defer branch.Free()

	err = repo.CheckoutBranch(branch)
	if err != nil {
		log.Fatalf("could not checkout branch: %s", err.Error())
	}
}

//==================================================
// create action
//==================================================
func action_create_package(ctx *cli.Context) {
	// check our args are in bound, break them out in vars for later flexibility
	max_args := 1
	num_args := len(ctx.Args())
	if num_args > max_args {
		log.Fatalf("too many arguments/packages (%d), max=%d", len(ctx.Args()), max_args)
	} else if num_args == 0 {
		log.Fatal("no package name provided")
	}

	// get the config to read the repo directory
	repo, err := repository.Open()
	if err != nil {
		log.Fatalf("could not open repository: %s", err.Error())
	}

	// check the package does not already exist
	package_name := ctx.Args()[0]
	package_path := path.Join(repo.Path, package_name)
	if strings.HasPrefix(package_path, "~/") {
		package_path = path.Join(os.Getenv("HOME"), package_path[2:])
	}

	if _, err := os.Stat(package_name); err == nil {

	}

	// create the dir
	err = os.Mkdir(package_path, 0755) // TODO: right perms?
	if err != nil {
		log.Fatalf("could not create package: %s", err.Error())
	}

	new_pkg := pkg.Info{
		Name: package_name,
	}

	install_target := ctx.String("target")
	cmd := ctx.String("cmd")
	pre_cmd := ctx.String("pre")
	post_cmd := ctx.String("post")

	// check if we were given a target (mutually exclusive with install commands
	// if not a target option, check for install commands
	if install_target != "" {
		home_path := os.Getenv("HOME")
		if strings.HasPrefix(install_target, home_path) {
			new_pkg.Target = path.Join("~", install_target[len(home_path):])
		} else {
			new_pkg.Target = install_target
		}
	} else if cmd != "" {
		new_pkg.InstallCmd.Cmd = cmd

		if pre_cmd != "" {
			new_pkg.InstallCmd.PreCmd = pre_cmd
		}

		if post_cmd != "" {
			new_pkg.InstallCmd.PostCmd = post_cmd
		}
	}

	// add package to config's map and write it out
	// no state to cleanup on disk if writing config fails
	repo.Config.Packages[package_name] = new_pkg
	if err := repo.Config.Write(path.Join(repo.Path, config.Name)); err != nil {
		log.Fatalf("could not write config after adding package: %s", err.Error())
	}

	// create a file if asked
	if file_name := ctx.String("file"); file_name != "" {
		file_path := path.Join(package_path, file_name)
		f, err := os.Create(file_path)
		if err != nil {
			log.Fatalf("could not create initial file: %s", file_path)
		}

		// if we need to make exec...
		if ctx.Bool("exec") {
			// get current perms. could be os dependent on creation
			s, err := f.Stat()
			if err != nil {
				log.Fatalf("error getting file data after creation: %s", err.Error())
			}

			// OR the existing mode with executable for all users
			// TODO: for all users?
			if err := f.Chmod(s.Mode() | os.FileMode(0111)); err != nil {
				log.Fatalf("could not makefile executable: %s", err.Error())
			}
		}

		if ctx.Bool("edit") {
			editor := os.ExpandEnv("$EDITOR")
			if len(editor) > 0 {
				editor_path, err := exec.LookPath(editor)
				if err != nil {
					log.Fatalf("could not find %s: %s", editor, err.Error())
				}

				cmd := exec.Command(editor_path, file_path)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Run()
				if err != nil {
					log.Fatalf("error while running %s: %s", editor_path, err.Error())
				}
			} else {
				log.Printf("WARN: no default editor in environment -- skipping.")
			}
		}
	}
}

//==================================================
// remove action
//==================================================
func action_remove_package(ctx *cli.Context) {
	conf, err := config.Open()
	if err != nil {
		log.Fatal(err)
	}

	args := ctx.Args()
	if len(args) == 0 {
		log.Fatalf("no package name given.")
	}

	for _, p := range args {
		if _, exists := conf.Packages[p]; exists == false {
			log.Printf("pakage %s does not exist", p)
			continue
		}

		dir := path.Join(conf.BaseDirectory, p)
		err = os.RemoveAll(dir)
		if err != nil {
			log.Fatal(err)
		}

		delete(conf.Packages, p)
	}

	if err := conf.Write(config.Path()); err != nil {
		log.Fatal(err)
	}
}

//==================================================
// modify action
//==================================================
func action_modify_package(ctx *cli.Context) {
	conf, err := config.Open()
	if err != nil {
		log.Fatal(err)
	}

	args := ctx.Args()
	if len(args) == 0 {
		log.Fatalf("no package name given.")
	}

	for _, p := range args {
		pack, exists := conf.Packages[p]
		if exists == false {
			log.Printf("pakage %s does not exist", p)
			continue
		}

		target := ctx.String("target")
		cmd := ctx.String("cmd")
		pre_cmd := ctx.String("pre")
		post_cmd := ctx.String("post")

		if len(target) > 0 {
			home_path := os.Getenv("HOME")
			if strings.HasPrefix(target, home_path) {
				pack.Target = path.Join("~", target[len(home_path):])
			} else {
				pack.Target = target
			}
		} else if cmd != "" {
			pack.InstallCmd.Cmd = cmd

			if pre_cmd != "" {
				pack.InstallCmd.PreCmd = pre_cmd
			}

			if post_cmd != "" {
				pack.InstallCmd.PostCmd = post_cmd
			}
		}

		conf.Packages[p] = pack
	}

	if err := conf.Write(config.Path()); err != nil {
		log.Fatal(err)
	}
}

//==================================================
// install action
//==================================================
func action_install(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) == 0 {
		log.Fatalf("no package name given.")
	}

	repo, err := repository.Open()
	if err != nil {
		log.Fatal(err)
	}

	for _, p := range args {
		pack, exists := repo.GetPackage(p)
		if exists == false {
			log.Fatalf("cannot install unknown package: %s", p)
		}

		fmt.Printf("[ install ] %s  to  ", p)
		err := pack.Install(path.Join(repo.Path, p))
		if err != nil {
			log.Fatal(err) // TODO: allow skipping errors
		}
	}
}

//==================================================
// update action
//==================================================
func action_update(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) == 0 {
		log.Fatalf("no package name given.")
	}

	repo, err := repository.Open()
	if err != nil {
		log.Fatal(err)
	}

	for _, p := range args {
		pack, exists := repo.GetPackage(p)
		if exists == false {
			log.Fatalf("cannot update unknown package: %s", p)
		}

		fmt.Printf("[ update ] %s... ", p)
		err := pack.Update(path.Join(repo.Path, p))
		if err != nil {
			log.Fatal(err) // TODO: allow skipping errors
		}
		fmt.Printf("done!\n")
	}
}

//==================================================
// pull action
//==================================================
func action_pull(ctx *cli.Context) {
	repo, err := repository.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Free()

	err = repo.Pull()
	if err != nil {
		log.Fatal(err)
	}

	// get what was changed so we can run callbacks
	changed, err := repo.ChangedInLastCommit()
	if err != nil {
		log.Fatal(err)
	}

	cache := make(map[string]bool)

	// iterate them
	for _, changed_path := range changed {
		// get the relpath of the path that was changed
		pkg_name, err := filepath.Rel(repo.Path, changed_path)
		if err != nil {
			continue
		}

		// this relpath may be a file or down deep in the tree
		// but strip it down to the package name
		pkg_name = filepath.SplitList(pkg_name)[0]
		pkg_name = strings.TrimRight(pkg_name, "/") // TODO: get generic separator

		// skip if we have run the commands for this package already
		if _, cached := cache[pkg_name]; cached {
			continue
		}
		cache[pkg_name] = true

		// skip this if not a package
		pack, exists := repo.GetPackage(pkg_name)
		if exists == false {
			continue
		}

		// take either install or update action based on the created or
		// modified status of the package in the commit
		if repo.CreatedInLast(changed_path) && ctx.IsSet("install") {
			err := pack.Install(path.Join(repo.Path, pkg_name))
			if err != nil {
				// TODO: give arg to not fatal on error
				log.Fatal(err)
			}
		} else if ctx.IsSet("upgrade") {
			err := pack.Update(path.Join(repo.Path, pkg_name))
			if err != nil {
				// TODO: give arg to not fatal on error
				log.Fatal(err)
			}
		}
	}

}

//==================================================
// upgrade action
//==================================================
func action_upgrade(ctx *cli.Context) {
	panic("upgrade command not currently supported. Will be included after PR: https://github.com/codegangsta/cli/pull/234")
}

//==================================================
// save action
//==================================================
func action_save(ctx *cli.Context) {
	repo, err := repository.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Free()

	msg := ctx.String("message")
	c, err := repo.CommitAll(msg)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Free()

	if ctx.IsSet("no-push") == false {
		err = repo.Push("master") // TODO: not only master
		if err != nil {
			log.Fatal(err)
		}
	}
}

//==================================================
// tag action
//==================================================
func action_tag(ctx *cli.Context) {
	panic("tag command not implemented")
}
