package repository

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/zmarcantel/hearth/config"

	git "github.com/libgit2/git2go"
)

func DefaultPath() string {
	return path.Join(os.Getenv("HOME"), ".hearth")
}

type Repository struct {
	Path string
	Repo *git.Repository
}

func Open() (Repository, error) {
	var repo Repository
	conf, err := config.Open()
	if err != nil {
		return repo, err
	}

	repo.Path = conf.BaseDirectory
	repo.Repo, err = git.OpenRepository(repo.Path) // use our new repo to assert good state
	if err != nil {
		return repo, fmt.Errorf("could not open git repository: %s", err)
	}

	return repo, nil
}

func Create(path, origin string) (Repository, error) {
	var err error
	var repo Repository
	if _, err = os.Stat(path); err == nil {
		return repo, fmt.Errorf("%s already exists.", path)
	}

	repo.Path = path
	repo.Repo, err = git.InitRepository(repo.Path, false)

	if err := InitFiles(repo); err != nil {
		return repo, err
	}

	if len(origin) == 0 {
		fmt.Println("WARN: origin not provided. must be added before issuing a save command")
	} else {
		_, err := repo.Repo.Remotes.Create("origin", origin)
		if err != nil {
			log.Fatalf("could not add remote:origin to repo: %s", err.Error())
		}
	}

	return repo, nil
}

func InitFiles(repo Repository) error {
	var conf config.Config
	conf.BaseDirectory = repo.Path
	config_path := path.Join(repo.Path, config.Name)

	return config.Write(config_path, conf)
}
