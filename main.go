package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/zmarcantel/hearth/config"
	"github.com/zmarcantel/hearth/repository"

	"github.com/codegangsta/cli"
)

func main() {
	app := init_flags()
	if err := app.Run(os.Args); err != nil {
		log.Fatalf(err.Error())
	}
}

func print_install(indent string, conf config.InstallConfig) {
	if conf.PreCmd != "" {
		fmt.Printf("%s- %s\n", indent, conf.PreCmd)
	}

	if conf.Cmd != "" {
		fmt.Printf("%s- %s\n", indent, conf.Cmd)
	}

	if conf.PostCmd != "" {
		fmt.Printf("%s- %s\n", indent, conf.PostCmd)
	}
}

//==================================================
// default action
//==================================================

// TODO: what should this be?
func action_default(ctx *cli.Context) {
	// load the config
	conf, err := config.Open()
	if os.IsNotExist(err) {
		log.Fatalf("failed to load hearth config from [%s], please use the create command to make one", config.Path())
	} else if err != nil {
		log.Fatalf("could not read/load config file: %s", err.Error())
	}

	// have config
	for name, env := range conf.Environments {
		fmt.Printf("Installing Environment: %s\n", name)
		print_install("\t", env.Install)
	}

	// TODO: better default action!!!
	for name, app := range conf.Configs {
		fmt.Printf("Installing App: %s\n", name)
		print_install("\t", app.Install)
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
	conf, err := config.Open()
	if err != nil {
		log.Fatalf("could not open config: %s", err.Error())
	}

	// check the package does not already exist
	package_name := ctx.Args()[0]
	package_path := path.Join(conf.BaseDirectory, package_name)
	if _, err := os.Stat(package_name); err == nil {

	}

	// create the dir
	err = os.Mkdir(package_path, 0755) // TODO: right perms?
	if err != nil {
		log.Fatalf("could not create package: %s", err.Error())
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
	}
}

//==================================================
// install action
//==================================================
func action_install(ctx *cli.Context) {
	panic("install command not implemented")
}

//==================================================
// install action
//==================================================
func action_update(ctx *cli.Context) {
	panic("update command not implemented")
}

//==================================================
// pull action
//==================================================
func action_pull(ctx *cli.Context) {
	panic("pull command not implemented")
}

//==================================================
// upgrade action
//==================================================
func action_upgrade(ctx *cli.Context) {
	panic("upgrade command not implemented")
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
