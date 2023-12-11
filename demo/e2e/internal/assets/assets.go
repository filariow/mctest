package assets

import _ "embed"

//go:embed config/cluster/kind/default-host-cluster.yaml
var DefaultClusterSpec string
