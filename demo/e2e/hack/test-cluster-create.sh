#!/bin/bash

set -e -o pipefail # -x

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

source "$SCRIPT_DIR/colors.sh"
source "$SCRIPT_DIR/test-cluster-common.sh"

create_cluster()
{
    print_section "creating management cluster"
    local _init_pid
    init &
    _init_pid=$!

    # create cluster and wait for its creation
    $KIND create cluster --name "$CLUSTER_NAME" --config "$KIND_CONFIG"

    # install clusterapi and exuma operators
    CLUSTER_TOPOLOGY=true $CLUSTERCTL init --infrastructure=docker --wait-providers &
    $MAKE --directory member install &
    # $MAKE --directory host/balancer install &
    $MAKE --directory host/cluster-metrics install &
    $MAKE --directory host/core install &
    $MAKE --directory host/space-foundation install &

    wait $_init_pid
    reload_images &

    wait
}

print_title "Running ${BASH_SOURCE[0]} $*"
create_cluster
