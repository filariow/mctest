#!/bin/bash

set -e -o pipefail # -x

CLEANUP_SCRIPT="test-cluster-cleanup.sh"
CREATE_SCRIPT="test-cluster-create.sh"
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

source "$SCRIPT_DIR/colors.sh"
source "$SCRIPT_DIR/test-cluster-common.sh"

create()
{
    echo "cluster $CLUSTER_NAME does NOT exists: creating"
    bash -c "time ${SCRIPT_DIR}/$CREATE_SCRIPT"
}

cleanup()
{
    echo "cluster $CLUSTER_NAME exists: trying to cleanup"
    bash -c "timeout $TIMEOUT time ${SCRIPT_DIR}/$CLEANUP_SCRIPT"
}

main()
{
    print_title "Running ${BASH_SOURCE[0]} $*"

    if $KIND get clusters | grep "$CLUSTER_NAME" > /dev/null; then
        if cleanup; then
            return 0
        fi

        $KIND delete cluster --name "$CLUSTER_NAME"
    fi

    create
}

main "$@"
