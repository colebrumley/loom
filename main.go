package main

import (
	"github.com/codegangsta/cli"
	"github.com/colebrumley/dockeraction"
	"github.com/docker/libkv/store"
	"github.com/fsouza/go-dockerclient"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"
)

type DockerExecProfile struct {
	Config     *docker.Config
	HostConfig *docker.HostConfig
}

var (
	kvStore       store.Store
	dockerClient  *dockeraction.ActionClient
	baseKey       string
	myHostname    string
	logger        *log.Logger
	WeaveProfiles = map[string]DockerExecProfile{
		"ps": DockerExecProfile{
			HostConfig: &docker.HostConfig{
				Privileged:  true,
				NetworkMode: "host",
				Binds:       []string{"/var/run/docker.sock:/var/run/docker.sock", "/proc:/hostproc"},
			},
			Config: &docker.Config{
				Image: "weaveworks/weaveexec:1.0.2",
				Env:   []string{"PROCFS=/hostproc"},
				Cmd:   []string{"--local", "ps"},
			},
		},
	}
)

func init() {
	logger = log.New()
	h, err := os.Hostname()
	if err != nil {
		logger.Fatal(err)
	}
	myHostname = h
	baseKey = "network/weave/" + myHostname + "/"
	dockerClient, err = dockeraction.GetDefaultActionClient()
	if err != nil {
		logger.Fatal(err)
	}
}

func main() {
	app := cli.NewApp()
	app.Version = "v0.1"
	app.Name = "Loom"
	app.Usage = "Weave KV Bridge"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "kvtype",
			Usage:  "KV Backend type (consul, etcd, zk)",
			Value:  "consul",
			EnvVar: "KV_TYPE",
		},
		cli.StringSliceFlag{
			Name:   "kvurl",
			Usage:  "KV Store endpoint",
			Value:  &cli.StringSlice{"127.0.0.1:8500"},
			EnvVar: "KV_URL_LIST",
		},
		cli.BoolFlag{
			Name:  "rm",
			Usage: "Remove previous values before updating",
		},
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "Enable verbose (debug) output",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:   "onetime",
			Usage:  "Update KV backend and exit",
			Action: oneTime,
		},
		{
			Name:   "daemon",
			Usage:  "Continuously update KV backend",
			Action: daemonize,
		},
	}
	app.Run(os.Args)
}

func daemonize(c *cli.Context) {
	// Start by updating current container list
	oneTime(c)
	eventChan := make(chan *docker.APIEvents)
	logger.Infoln("Watching for Docker events")
	if err := dockerClient.AddEventListener(eventChan); err != nil {
		logger.Fatal(err)
	}

	for event := range eventChan {
		event := event
		if !strings.Contains(event.From, "weaveexec") && (event.Status == "start" || event.Status == "die") {
			go func() {
				logger.Debugf("Received Docker event %s from %s (%s)", event.Status, event.From, event.ID[:12])
				switch event.Status {
				case "start":
					for i := 0; i < 20; i++ {
						logger.Debugf("Scanning Weave for %s", event.ID[:12])
						for _, w := range runWeavePs() {
							if event.ID[:12] == w.ID {
								if !kvWeaveExists(w) {
									logger.Infof("Adding %s to KV store as %s", w.ID, w.Name)
									if err := registerWeaveIPToKV(w); err != nil {
										logger.Error(err)
									}
								} else {
									logger.Infof("Skipping " + w.Name + " because it's already is the KV store.")
								}
								return
							}
						}
						logger.Debugf("No match found in Weave for %s", event.ID[:12])
						time.Sleep(500 * time.Millisecond)
						i++
					}
				case "die":
					eid := event.ID[:12]
					logger.Infof("Removing %s from KV", eid)
					rm, err := kvRmExists(eid)
					if rm {
						logger.Infof("Removed %s from KV", eid)
					} else {
						logger.Infof("Failed to remove %s: %v", eid, err)
					}
				}
			}()
		}
	}
}

func oneTime(c *cli.Context) {
	if c.GlobalBool("verbose") {
		logger.Level = log.DebugLevel
	}
	initializeKVStore(c)
	if c.GlobalBool("rm") {
		logger.Infoln("Removing previous entries from KV")
		if _, err := kvStore.List(baseKey); err == nil {
			kvStore.DeleteTree(baseKey)
		}
	}
	// Run through weave's initial state
	logger.Infoln("Scanning Weave")
	for _, w := range runWeavePs() {
		logger.Debugf(
			"Found Weave address for %s:\n  IP:\t%s\n  CIDR:\t%s\n  ID:\t%s\n  MAC:\t%s",
			w.Name, w.IP, w.CIDR, w.ID, w.MAC)
		if !kvWeaveExists(w) {
			logger.Infof("Adding %s to KV store", w.Name)
			if err := registerWeaveIPToKV(w); err != nil {
				logger.Fatal(err)
			}
		} else {
			logger.Infof("Skipping " + w.Name + " because it's already is the KV store.")
		}
	}
	logger.Infoln("Finished scanning Weave")
}

func setName(w *WeaveIP) {
	if w.ID != "weave:expose" {
		container, err := dockerClient.InspectContainer(w.ID)
		if err != nil {
			logger.Fatal(err)
		}
		w.Name = strings.TrimPrefix(container.Name, "/")
	} else {
		w.Name = strings.TrimPrefix(w.ID, "weave:")
	}
}

func runWeavePs() []*WeaveIP {
	results := []*WeaveIP{}
	out, err := dockerClient.RunOutput(WeaveProfiles["ps"].Config, WeaveProfiles["ps"].HostConfig, true)
	if err != nil {
		logger.Error(err)
		return results
	}

	// parse weave ps command output
	// sample:
	//   9a000157e68a c6:7a:e5:41:39:b8 10.2.0.2/16
	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		columns := strings.Split(l, " ")
		if len(columns) < 3 {
			continue
		}
		wip := &WeaveIP{
			ID:   columns[0],
			MAC:  columns[1],
			IP:   strings.Split(columns[2], "/")[0],
			CIDR: strings.Split(columns[2], "/")[1],
		}
		logger.Debugf("Parsed ID: %s, IP: %s, MAC: %s, CIDR: %s", wip.ID, wip.IP, wip.MAC, wip.CIDR)
		setName(wip)
		results = append(results, wip)
	}
	return results
}
