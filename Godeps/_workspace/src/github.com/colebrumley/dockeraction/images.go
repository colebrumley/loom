package dockeraction

import (
	"errors"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"io"
	"net/url"
)

// PullImageWithDefaults uses the standard ~/.dockercfg auth file to pull an image.
// Does not currently work with the docker-machine style ~/.docker/config.json file.
func (l *ActionClient) PullImageWithDefaults(repo, image, tag string, outputStream io.Writer) error {
	if repo == "" {
		repo = "index.docker.io"
	}
	auth, err := docker.NewAuthConfigurationsFromDockerCfg()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}

	for _, v := range auth.Configs {
		repoUrl, err := url.Parse(v.ServerAddress)
		if err != nil {
			continue
		}
		if repoUrl.Host == repo || v.ServerAddress == repo {
			return l.PullImage(docker.PullImageOptions{
				OutputStream: outputStream,
				Repository:   repo + "/" + image,
				Tag:          tag,
			}, v)
		}
	}
	return errors.New("Could not find docker auth")
}

// ImageExists returns a bool indicating whether the image
// is present is the current Docker daemon
func (l *ActionClient) ImageExists(repotag string) bool {
	imgs, err := l.ListImages(docker.ListImagesOptions{})
	if err != nil {
		return false
	}
	for _, i := range imgs {
		for _, t := range i.RepoTags {
			if t == repotag {
				return true
			}
		}
	}
	return false
}
