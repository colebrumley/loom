# loom
[![Circle CI](https://circleci.com/gh/colebrumley/loom.svg?style=svg)](https://circleci.com/gh/colebrumley/loom)

Register Weave network info to a key-value backend using docker's libkv for use in service discovery and configuration.

**Key structure:**
- `network/weave/[hostname]/[container_name]/[id|ip|mac|cidr]`

**Notes:**
- `weave expose` is registered under the name `expose` and id `weave:expose`.  MAC, CIDR, and IP are standard.
- Only *local* Weave addresses are registered.
