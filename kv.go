package main

import (
	"github.com/codegangsta/cli"
	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"strings"
)

type WeaveIP struct {
	ID   string
	Name string
	MAC  string
	IP   string
	CIDR string
}

func initializeKVStore(c *cli.Context) {
	var kvt store.Backend
	switch c.GlobalString("kvtype") {
	case "consul":
		kvt = store.CONSUL
	case "etcd":
		kvt = store.ETCD
	case "zk":
		kvt = store.ZK
	}
	k, err := libkv.NewStore(kvt, c.GlobalStringSlice("kvurl"), &store.Config{})
	if err != nil {
		logger.Fatal(err)
	}
	kvStore = k
}

func kvWeaveExists(w *WeaveIP) bool {
	id, err := kvStore.Get(baseKey + w.Name + "/id")
	if err == nil && string(id.Value) == w.ID {
		return true
	}
	return false
}

func kvRmExists(id string) (rm bool, err error) {
	rm = false
	pairs, err := kvStore.List(baseKey)
	if err != nil {
		return
	}
	for _, p := range pairs {
		if string(p.Value) == id[:12] {
			path := strings.Split(p.Key, "/")
			err = kvStore.Delete(baseKey + path[len(path)-2] + "/ip")
			err = kvStore.Delete(baseKey + path[len(path)-2] + "/cidr")
			err = kvStore.Delete(baseKey + path[len(path)-2] + "/id")
			err = kvStore.Delete(baseKey + path[len(path)-2] + "/mac")
			rm = true
			return
		}
	}
	return
}

func registerWeaveIPToKV(w *WeaveIP) error {
	key := baseKey + w.Name + "/"
	if err := kvStore.Put(key+"ip", []byte(w.IP), &store.WriteOptions{}); err != nil {
		return err
	}
	if err := kvStore.Put(key+"mac", []byte(w.MAC), &store.WriteOptions{}); err != nil {
		return err
	}
	if err := kvStore.Put(key+"id", []byte(w.ID), &store.WriteOptions{}); err != nil {
		return err
	}
	return kvStore.Put(key+"cidr", []byte(w.CIDR), &store.WriteOptions{})
}
