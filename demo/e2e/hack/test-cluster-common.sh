#!/bin/bash

[ -z "$CLUSTERCTL" ] && export CLUSTERCTL="clusterctl"
[ -z "$KIND" ] && export KIND="kind"
[ -z "$KIND_CONFIG" ] && export KIND_CONFIG="e2e/config/management-cluster/kind/kind-cluster-with-extramounts.yaml"
[ -z "$CLUSTER_NAME" ] && export CLUSTER_NAME="exuma-e2e-mgmt"
[ -z "$KUBECTL" ] && export KUBECTL="kubectl"
[ -z "$TIMEOUT" ] && export TIMEOUT="120s"
[ -z "$MAKE" ] && export MAKE="make"
[ -z "$MAKE_ARGS" ] && export MAKE_ARGS="-j4"

init()
{
    # prepare, test base folder and build images
    print_section "preparing manifests and images"
    rm -rf '.tmp'
    $MAKE e2e-prepare-prebase-folder build-images || { print_error "error initializing preparing base folder and/or building images"; exit 1; }
}

reload_images()
{
    print_section "reloading exuma images into management cluster"
    $KIND load docker-image --name "$CLUSTER_NAME" exuma/member:test-latest &
    # $KIND load docker-image --name "$CLUSTER_NAME" exuma/balancer:test-latest &
    $KIND load docker-image --name "$CLUSTER_NAME" exuma/cluster-metrics:test-latest &
    $KIND load docker-image --name "$CLUSTER_NAME" exuma/core:test-latest &
    $KIND load docker-image --name "$CLUSTER_NAME" exuma/space-foundation:test-latest &
    wait || { print_error "error loading exuma's docker images into management cluster"; exit 1; }
}
