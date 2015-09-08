package dockeraction

import (
	"github.com/fsouza/go-dockerclient"
	"testing"
)

func TestGetDefaultActionClient(t *testing.T) {
	if _, err := GetDefaultActionClient(); err != nil {
		if err.Error() == "No valid Docker endpoint found" {
			t.Skipf("No docker endpoint found")
		}
		t.Errorf("Failed to get client: %v", err)
	}
}

func TestGetContainerFromName(t *testing.T) {
	client, err := GetDefaultActionClient()
	if err != nil {
		t.Skipf("No docker client to test against")
	}
	container, err := client.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: "alpine",
			Cmd:   []string{"ifconfig"},
		},
		Name: "testimage",
	})
	if err != nil {
		t.Errorf("%v", err)
	}
	if err := client.StartContainer(container.ID, &docker.HostConfig{}); err != nil {
		t.Errorf("%v", err)
	}
	test, err := client.GetContainerFromName("testimage")
	if err != nil {
		t.Errorf("%v", err)
	}
	if test.ID != container.ID {
		t.Errorf("GetContainerFromName got a different ID than expected")
	}
	client.RemoveContainer(docker.RemoveContainerOptions{
		ID:    container.ID,
		Force: true,
	})
}
