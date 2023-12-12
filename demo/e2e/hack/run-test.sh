#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

source "$SCRIPT_DIR/colors.sh"
source "$SCRIPT_DIR/test-cluster-common.sh"

GODOG_CONCURRENCY="${GODOG_CONCURRENCY:-1}"
[ -n "$GODOG_WIP" ] && GODOG_TAGS="--godog.tags=wip"
GODOG_TAGS=${GODOG_TAGS:-"--godog.tags=~disabled"}

# GODOG_EXTRA_ARGS="${GODOG_EXTRA_ARGS:-}"

run_tests()
{
    print_title "Running e2e tests"
    td=$(realpath $SCRIPT_DIR/..)
    set -x

    go -C "$td" test -v \
        "$GODOG_TAGS" \
        --godog.concurrency "$GODOG_CONCURRENCY" \
        "$GODOG_EXTRA_ARGS"
    set +x
}

bash -c "${SCRIPT_DIR}/start-or-clean-kind.sh" && run_tests
