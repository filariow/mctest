#!/bin/bash

set -e -o pipefail # -x

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

source "$SCRIPT_DIR/colors.sh"
source "$SCRIPT_DIR/test-cluster-common.sh"

try_cleanup()
{
    local _init_pid
    init &
    _init_pid=$!

    print_section "cleaning up management cluster"
    # store admin kubeconfig to management cluster in a temp file
    local kf
    kf=$(mktemp)

    $KIND get kubeconfig --name "$CLUSTER_NAME" > "$kf" || exit 1

    # delete all clusters
    $KUBECTL delete "clusters.cluster.x-k8s.io" --all --all-namespaces --wait --kubeconfig "$kf" --ignore-not-found=true || \
        { print_error "error deleting clusters " && exit 1; }

    # delete other clusterapi resources
    $KUBECTL api-resources --verbs=list --namespaced -o name --kubeconfig "$kf" | \
            grep "x-k8s.io" | \
            grep -Ev "clusters.cluster.x-k8s.io" | \
            grep -Ev "providers.clusterctl.cluster.x-k8s.io" | \
            xargs -P 4 -I @ kubectl delete @ --all --all-namespaces --wait --kubeconfig "$kf" || \
            { print_error "error deleting cluster-api resources" && exit 1; } &

    # delete namespaces
    $KUBECTL delete namespaces --wait -l scope=test --kubeconfig "$kf" --ignore-not-found=true || \
        { print_error "error deleting namespaces" && exit 1; } &

    wait $_init_pid
    reload_images &

    wait
}

print_title "Running ${BASH_SOURCE[0]} $*"
try_cleanup
