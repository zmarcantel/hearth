package main

import (
	"math/rand"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	git "github.com/libgit2/git2go"
)

func create_origin_repo(t *testing.T) string {
	rand.Seed(time.Now().UnixNano())
	repo_path := path.Join(os.TempDir(), strconv.FormatUint(uint64(rand.Int63()), 10))

	_, err := git.InitRepository(repo_path, true)
	if err != nil {
		t.Fatal(err)
	}

	return repo_path
}
