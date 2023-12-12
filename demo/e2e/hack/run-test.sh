#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
TESTS_DIR=$(realpath "$SCRIPT_DIR/..")

source "$SCRIPT_DIR/colors.sh"
source "$SCRIPT_DIR/test-cluster-common.sh"

GODOG_CONCURRENCY="${GODOG_CONCURRENCY:-1}"
[ -n "$GODOG_WIP" ] && GODOG_TAGS="--godog.tags=wip"
GODOG_TAGS=${GODOG_TAGS:-"--godog.tags=~disabled"}

run_tests()
{
    print_title "Running e2e tests"
    set -x

    go -C "$TESTS_DIR" test -v \
        "$GODOG_TAGS" \
        --godog.concurrency "$GODOG_CONCURRENCY"
}

go -C "$TESTS_DIR" vet ./... && \
    bash -c "${SCRIPT_DIR}/start-or-clean-kind.sh" && \
    run_tests
