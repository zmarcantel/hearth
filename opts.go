package main

import (
	"fmt"
	"net"
	"os"

	"github.com/codegangsta/cli"
)

const (
	HEARTH_VERSION_MAJOR = 0
	HEARTH_VERSION_MINOR = 1
	HEARTH_VERSION_PATCH = 0
)

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
		// daemon
		//==================================================
		{
			Name:        "daemon",
			Usage:       "start the repo-watching daemon",
			Description: "start the repo-watching daemon",
			Flags: []cli.Flag{
				cli.GenericFlag{
					Name:  "file",
					Value: FileFlag{},
					Usage: "path to the file which will be used as IPC",
				},
				cli.GenericFlag{
					Name:  "socket",
					Value: SocketFlag{},
					Usage: "address from which to serve daemon API calls",
				},
			},
			Action: start_daemon,
		},
	}

	app.Action = start_cli

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
