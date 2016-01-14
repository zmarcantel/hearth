package main

import (
	"fmt"
	"net"
	"os"
	"path"

	"github.com/codegangsta/cli"
)

const (
	HEARTH_VERSION_MAJOR = 0
	HEARTH_VERSION_MINOR = 1
	HEARTH_VERSION_PATCH = 0
)

type Options struct {
	// working environment
	ConfigFile string
	ConfigPath string
	RepoPath   string

	// package creation
	StartWithEditor bool
	InitPackageFile string
	InitPackageExec bool

	// package selection
	AllPackages     bool
	AllEnvironments bool
	AllAppConfigs   bool
	PackageList     []string
	PackageRegex    string

	// pull actions
	InstallNewPackages bool
	UpdateAfterPull    bool

	// save options
	SkipPush      bool
	CommitMessage string
}

var opts Options

//==================================================
// setup all our flags and route subcommands
//==================================================

func init_flags() *cli.App {
	app := cli.NewApp()
	app.Name = "hearth"
	app.Usage = "settings and dotfiles made easy"
	app.HideHelp = true
	app.Version = fmt.Sprintf("%d.%d.%d", HEARTH_VERSION_MAJOR, HEARTH_VERSION_MINOR, HEARTH_VERSION_PATCH)
	app.Authors = []cli.Author{
		{Name: "Zach Marcantel", Email: "zmarcantel@gmail.com"},
	}

	app.Commands = []cli.Command{
		//==================================================
		// init
		//==================================================
		{
			Name:        "init",
			Usage:       "creates a config file and initial git repository (interactive unless -p/-f/-r supplied)",
			Description: "creates a config file and initial git repository (interactive unless -p/-f/-r supplied)",
			Flags:       []cli.Flag{},
			Action:      action_create_config,
		},
		//==================================================
		// create
		//==================================================
		{
			Name:        "create",
			Usage:       "create a new package, and optionally start $EDITOR",
			Description: "create a new package, and optionally start $EDITOR",
			Action:      action_create_package,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:        "edit",
					Usage:       "open the given file [default: main.sh] in the new package",
					Destination: &opts.StartWithEditor,
				},
				cli.StringFlag{
					Name:        "file",
					Usage:       "name of the main file to create in the package",
					Destination: &opts.InitPackageFile,
				},
				cli.BoolFlag{
					Name:        "exec",
					Usage:       "make the file created as defined by --file executable",
					Destination: &opts.InitPackageExec,
				},
			},
		},
		//==================================================
		// install
		//==================================================
		{
			Name:        "install",
			Usage:       "install one or many packages",
			Description: "install one or many packages",
			Action:      action_install,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:        "all",
					Usage:       "install all packages listed in the config",
					Destination: &opts.AllPackages,
				},
				cli.BoolFlag{
					Name:        "environments",
					Usage:       "install all environments listed in the config",
					Destination: &opts.AllEnvironments,
				},
				cli.BoolFlag{
					Name:        "apps",
					Usage:       "install all application configs listed in the config",
					Destination: &opts.AllAppConfigs,
				},
				cli.StringFlag{
					Name:        "filter",
					Usage:       "regular expression (go syntax) for packages to install",
					Destination: &opts.PackageRegex,
				},
			},
		},
		//==================================================
		// update
		//==================================================
		{
			Name:        "update",
			Usage:       "update one or many packages",
			Description: "update one or many packages",
			Action:      action_update,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:        "all",
					Usage:       "update all packages listed in the config",
					Destination: &opts.AllPackages,
				},
				cli.BoolFlag{
					Name:        "environments",
					Usage:       "update all environments listed in the config",
					Destination: &opts.AllEnvironments,
				},
				cli.BoolFlag{
					Name:        "apps",
					Usage:       "update all application configs listed in the config",
					Destination: &opts.AllAppConfigs,
				},
				cli.StringFlag{
					Name:        "filter",
					Usage:       "regular expression (go syntax) for packages to update",
					Destination: &opts.PackageRegex,
				},
			},
		},
		//==================================================
		// pull
		//==================================================
		{
			Name:        "pull",
			Usage:       "pull any changes from 'origin'",
			Description: "pull any changes from 'origin'",
			Action:      action_pull,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:        "install",
					Usage:       "install any new packages",
					Destination: &opts.InstallNewPackages,
				},
				cli.BoolFlag{
					Name:        "update",
					Usage:       "update all packages after pulling",
					Destination: &opts.UpdateAfterPull,
				},
			},
		},
		//==================================================
		// upgrade
		//==================================================
		{
			Name:        "upgrade",
			Usage:       "alias of 'pull --install --update'",
			Description: "alias of 'pull --install --update'",
			Action:      action_upgrade,
		},
		//==================================================
		// save
		//==================================================
		{
			Name:        "save",
			Usage:       "create a commit message (or use default) and push to 'origin'",
			Description: "create a commit message (or use default) and push to 'origin'",
			Action:      action_save,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:        "no-push",
					Usage:       "skip pushing to 'origin'",
					Destination: &opts.SkipPush,
				},
				cli.StringFlag{
					Name:        "m, message",
					Usage:       "use the given message",
					Destination: &opts.CommitMessage,
				},
			},
		},
		//==================================================
		// tag
		//==================================================
		{
			Name:        "tag",
			Usage:       "create a tag at the most recent commit",
			Description: "create a tag at the most recent commit",
			Action:      action_tag,
		},
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "f, config-file",
			Usage:       "use a name other than .hearthrc when looking for config",
			Value:       ".hearthrc",
			Destination: &opts.ConfigFile,
		},
		cli.StringFlag{
			Name:        "p, config-path",
			Usage:       "use a directory other than $HOME when looking for config",
			Value:       os.Getenv("HOME"),
			Destination: &opts.ConfigPath,
		},
		cli.StringFlag{
			Name:        "r, repo",
			Usage:       "use a directory other than $HOME/.hearth to contain the repo",
			Value:       path.Join(os.Getenv("HOME"), ".hearth"),
			Destination: &opts.RepoPath,
		},
	}

	app.Action = action_cli

	return app
}

//==================================================
// flag to get an open os.File
//==================================================
type FileFlag struct {
	File *os.File
}

func (f FileFlag) Set(value string) (err error) {
	f.File, err = os.OpenFile(value, os.O_RDWR, 600)
	return
}
func (f FileFlag) String() string {
	// handle the help string case where we have no value
	if f.File == nil {
		return "FILE"
	}

	// return actual
	return f.File.Name()
}

//==================================================
// flag to get an open net.TCPListener
//==================================================

type SocketFlag struct {
	Addr     *net.TCPAddr
	Listener *net.TCPListener
}

func (f SocketFlag) Set(value string) (err error) {
	f.Addr, err = net.ResolveTCPAddr("tcp", value)
	if err != nil {
		return
	}

	f.Listener, err = net.ListenTCP("tcp", f.Addr)
	return
}
func (f SocketFlag) String() string {
	// handle the help string case where we have no value
	if f.Addr == nil {
		return "HOST:ADDR"
	}

	// return actual
	return f.Addr.String()
}
