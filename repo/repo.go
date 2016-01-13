package repo

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/zmarcantel/hearth/config"
	git "gopkg.in/libgit2/git2go.v22"
)

type Repository struct {
	Path    string
	Config  config.HearthConfig
	GitRepo *git.Repository
}

func CreateWithConfig(path string, interactive bool) (Repository, error) {
	var err error
	var conf config.HearthConfig
	if interactive {
		conf, err = config.CreateInteractive()
		if err != nil {
			return Repository{}, err
		}
	} else {
		// TODO
	}

	return Create(&conf)
}

func Create(conf *config.HearthConfig) (repo Repository, err error) {
	if conf != nil {
		return Repository{}, errors.New("cannot create repo with nil config")
	}
	repo.Path = conf.BaseDirectory
	repo.Config = *conf // copy the config, as we only modify the repo from now on
	repo.GitRepo, err = git.InitRepository(repo.Path, false)

	return
}

func Open() (Repository, error) {
	var err error
	var repo Repository

	repo.Config, err = config.Open()
	if err != nil {
		return Repository{}, err
	}
	repo.Path = repo.Config.BaseDirectory
	if err != nil {
		return repo, err
	}
	repo.Path = strings.Replace(repo.Path, "~/", fmt.Sprintf("%s/", os.Getenv("HOME")), -1)
	fmt.Println(repo.Path)

	repo.GitRepo, err = git.OpenRepository(repo.Path)
	if err != nil {
		var exist_err error
		if _, exist_err = os.Stat(repo.Path); os.IsNotExist(exist_err) {
			var create_repo string
			fmt.Print("config file exists, but repo does not... create one? [y/n] ")
			if fmt.Scanln(&create_repo); strings.ToLower(create_repo)[0] == 'y' {
				// create
				fmt.Printf("creating repo in %s\n", repo.Path)
				repo.GitRepo, err = git.InitRepository(repo.Path, false)
				if err != nil {
					return repo, err
				}
			} else {
				return repo, err // return the base error
			}
		}
		return repo, err
	}

	return repo, nil
}
