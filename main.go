package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/zmarcantel/hearth/config"

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

func action_cli(ctx *cli.Context) {
	config_path := path.Join(opts.ConfigPath, opts.ConfigFile)

	// load the config
	conf, err := config.Load(config_path)
	if os.IsNotExist(err) {
		log.Fatalf("failed to load hearth config from [%s], please use the create command to make one", config_path)
	} else if err != nil {
		log.Fatalf("could not read/load config file: %s", err.Error())
	}

	// have config
	for name, env := range conf.Environments {
		fmt.Printf("Installing Environment: %s\n", name)
		print_install("\t", env.Install)
	}

	for name, app := range conf.Configs {
		fmt.Printf("Installing App: %s\n", name)
		print_install("\t", app.Install)
	}
}

func action_create_config(ctx *cli.Context) {
	if ctx.IsSet("config-path") || ctx.IsSet("config-file") || ctx.IsSet("repo") {
		_, err := config.Create(opts.ConfigPath, opts.ConfigFile, opts.RepoPath)
		if err != nil {
			log.Fatalf("could not create config: %s", err.Error())
		}
	} else {
		config_path := path.Join(opts.ConfigPath, opts.ConfigFile)
		_, err := config.CreateInteractive(config_path)
		if err != nil {
			log.Fatalf("could not create config: %s", err.Error())
		}
	}
}

func action_create_package(ctx *cli.Context) {
	panic("create_package command not implemented")
}

func action_install(ctx *cli.Context) {
	panic("install command not implemented")
}

func action_update(ctx *cli.Context) {
	panic("update command not implemented")
}

func action_pull(ctx *cli.Context) {
	panic("pull command not implemented")
}

func action_upgrade(ctx *cli.Context) {
	panic("upgrade command not implemented")
}

func action_save(ctx *cli.Context) {
	panic("save command not implemented")
}

func action_tag(ctx *cli.Context) {
	panic("tag command not implemented")
}
