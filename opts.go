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
	RepoPath   string
	RepoOrigin string

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
			Usage:       "creates a config file and initial git repository",
			Description: "creates a config file and initial git repository",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "r, repo",
					Usage:       "use a directory other than $HOME/.hearth to contain the repo",
					Value:       path.Join(os.Getenv("HOME"), ".hearth"),
					Destination: &opts.RepoPath,
				},
				cli.StringFlag{
					Name:        "o, origin",
					Usage:       "use the given URL/path as the git repo's origin (or, manually add later)",
					Value:       "",
					Destination: &opts.RepoOrigin,
				},
			},
			Action: action_init,
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
					Name:        "e, edit",
					Usage:       "open the given file [default: main.sh] in the new package",
					Destination: &opts.StartWithEditor,
				},
				cli.StringFlag{
					Name:        "f, file",
					Usage:       "create a file with the given name in the package",
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
			Usage:       "create a commit (or use default) and push to 'origin'",
			Description: "create a commit (or use default) and push to 'origin'",
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

	app.Action = action_default

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
