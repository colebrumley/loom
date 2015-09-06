package dockeraction

import (
	"bufio"
	"bytes"
	"github.com/fsouza/go-dockerclient"
	"strings"
	"time"
)

// RunOutput runs a container and blocks until it exits, capturing the
// output and optionally removing the container.
func (l *ActionClient) RunOutput(config *docker.Config, hostConfig *docker.HostConfig, rm bool) (out []byte, err error) {
	execContainer, err := l.CreateContainer(docker.CreateContainerOptions{
		Config:     config,
		HostConfig: hostConfig,
	})
	if err != nil {
		return
	}

	if err = l.StartContainer(execContainer.ID, hostConfig); err != nil {
		return
	}

	buf := bytes.NewBuffer([]byte{})
	if err = l.AttachToContainer(docker.AttachToContainerOptions{
		OutputStream: buf,
		Container:    execContainer.ID,
		Stdout:       true,
		Stderr:       true,
		Logs:         true,
		Stream:       true,
	}); err != nil {
		return
	}

	if _, err = l.WaitContainer(execContainer.ID); err != nil {
		return
	}
	if rm {
		go l.RemoveContainer(docker.RemoveContainerOptions{
			ID:    execContainer.ID,
			Force: true,
		})
	}
	out = buf.Bytes()
	return
}

// StreamLogs attaches to the provided container id and streams stderr/stdout
// to output and error channels. StreamLogs blocks until the container exits.
// To save CPU cycles, during periods of log inactivity StreamLogs sleeps for
// (5 * empty iterations) milliseconds up to maxSleep (2 seconds).
func (l *ActionClient) StreamLogs(id string, output chan string, errChan chan error, maxSleep ...int64) {
	buf := bytes.NewBuffer([]byte{})
	lineReader := bufio.NewReader(buf)
	kill := make(chan error, 1)

	go func() {
		kill <- l.AttachToContainer(docker.AttachToContainerOptions{
			OutputStream: buf,
			ErrorStream:  buf,
			Container:    id,
			Stdout:       true,
			Stderr:       true,
			Stream:       true,
			Logs:         true,
		})
	}()

	var (
		i     int64
		sleep int64
	)
	i = 0
	if len(maxSleep) > 0 {
		sleep = maxSleep[0]
	} else {
		sleep = 2000
	}
	for {
		select {
		// check for kill signal
		case err := <-kill:
			errChan <- err
			return
		default:
			l, err := lineReader.ReadString('\n')
			// We just want to skip if EOF and the
			// container is not dead
			if err != nil && err.Error() != "EOF" {
				errChan <- err
				return
			} else {
				// Don't clog the buffer with empty lines
				if len(l) > 0 {
					i = 0
					output <- strings.TrimSuffix(l, "\n")
				} else {
					i++
					if sleepVal := (5 * i); sleepVal < sleep {
						time.Sleep(time.Duration(sleepVal) * time.Millisecond)
					} else {
						time.Sleep(time.Duration(sleep) * time.Millisecond)
					}

				}
			}
		}
	}
}
