package dockeraction

import (
	"errors"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"os"
	"strings"
)

// ActionClient is the wrapper object for the Docker daemon.
type ActionClient struct {
	*docker.Client
}

// GetDefaultActionClient uses docker-machine style ENV vars to
// locate a docker host, falling back to /var/run/docker.sock if
// no vars are found
func GetDefaultActionClient() (client *ActionClient, err error) {
	var dockerClient *docker.Client
	// Get endpoint from env, then fallback to socket
	endpoint := ""
	if len(os.Getenv("DOCKER_HOST")) > 0 {
		endpoint = os.Getenv("DOCKER_HOST")
	} else if fileExists("/var/run/docker.sock") {
		endpoint = "unix:///var/run/docker.sock"
	} else {
		err = errors.New("No valid Docker endpoint found")
		return
	}

	// Get optional certs & connect
	if len(os.Getenv("DOCKER_CERT_PATH")) > 0 {
		path := os.Getenv("DOCKER_CERT_PATH")
		ca := fmt.Sprintf("%s/ca.pem", path)
		cert := fmt.Sprintf("%s/cert.pem", path)
		key := fmt.Sprintf("%s/key.pem", path)
		dockerClient, err = docker.NewTLSClient(endpoint, cert, key, ca)
	} else {
		dockerClient, err = docker.NewClient(endpoint)
	}
	if err != nil {
		return
	}
	client = &ActionClient{dockerClient}
	return
}

// GetActionClient gets an `*ActionClient` object.
func GetActionClient(endpoint, cert, key, ca string) (client *ActionClient, err error) {
	var dockerClient *docker.Client
	if len(cert) > 0 && len(key) > 0 && len(ca) > 0 {
		dockerClient, err = docker.NewTLSClient(endpoint, cert, key, ca)
	} else {
		dockerClient, err = docker.NewClient(endpoint)
	}
	if err != nil {
		return
	}
	client = &ActionClient{dockerClient}
	return
}

// Retrieves a `*docker.Container` object from a provided container name.
func (l *ActionClient) GetContainerFromName(name string) (c *docker.Container, err error) {
	apiContainers, err := l.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return
	}

	for _, container := range apiContainers {
		for _, thisName := range container.Names {
			if strings.TrimPrefix(thisName, "/") == name {
				c, err = l.InspectContainer(container.ID)
				return
			}
		}
	}
	return
}

func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
