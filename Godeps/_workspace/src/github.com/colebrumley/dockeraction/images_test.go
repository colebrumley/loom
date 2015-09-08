package dockeraction

import (
	"bytes"
	"os"
	"testing"
)

var (
	testImgs = map[string]map[string]string{
		"default": map[string]string{
			"repo": "index.docker.io",
			"img":  "elcolio/etcd",
			"tag":  "latest",
		},
		"default_empty_repo_name": map[string]string{
			"repo": "",
			"img":  "elcolio/etcd",
			"tag":  "latest",
		},
		"private": map[string]string{
			"repo": "docker-repo.gonkulator.io",
			"img":  "gonkulator/etcd",
			"tag":  "latest",
		},
	}
)

func TestPullImageWithDefaults(t *testing.T) {
	client, err := GetDefaultActionClient()
	if err != nil {
		t.Skipf("No docker client to test against")
	}
	// make sure docker auth is present
	if !fileExists(os.Getenv("HOME") + "/.dockercfg") {
		t.Skipf("No .dockercfg to test against")
	}

	buf := bytes.NewBuffer([]byte{})
	for name, cfg := range testImgs {
		if err := client.PullImageWithDefaults(cfg["repo"], cfg["img"], cfg["tag"], buf); err != nil {
			t.Errorf("Could not pull from %s repo: %v", name, err)
		}
	}
	if err := client.PullImageWithDefaults("", "i29cn9isdhf", "snarf", buf); err == nil {
		t.Errorf("Not supposed to be able to pull bad data...")
	}
}

func TestImageExists(t *testing.T) {
	client, err := GetDefaultActionClient()
	if err != nil {
		t.Skipf("No docker client to test against")
	}
	// make sure docker auth is present
	if !fileExists(os.Getenv("HOME") + "/.dockercfg") {
		t.Skipf("No .dockercfg to test against")
	}

	buf := bytes.NewBuffer([]byte{})
	client.PullImageWithDefaults(testImgs["default"]["repo"], testImgs["default"]["img"], testImgs["default"]["tag"], buf)
	if !client.ImageExists(testImgs["default"]["img"] + ":" + testImgs["default"]["tag"]) {
		t.Fatalf("Could not find " + testImgs["default"]["img"] + ":" + testImgs["default"]["tag"])
	}
	if client.ImageExists("3oncnoodhfks:kwcwodj") {
		t.Fatalf("Returned true for imageExists on a bad image name")
	}
}
