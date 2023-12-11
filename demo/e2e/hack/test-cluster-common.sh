#!/bin/bash

[ -z "$CLUSTERCTL" ] && export CLUSTERCTL="clusterctl"
[ -z "$KIND" ] && export KIND="kind"
[ -z "$KIND_CONFIG" ] && export KIND_CONFIG="e2e/config/management-cluster/kind-cluster-with-extramounts.yaml"
[ -z "$CLUSTER_NAME" ] && export CLUSTER_NAME="mctest-e2e-mgmt"
[ -z "$KUBECTL" ] && export KUBECTL="kubectl"
[ -z "$TIMEOUT" ] && export TIMEOUT="120s"
[ -z "$MAKE" ] && export MAKE="make"
[ -z "$MAKE_ARGS" ] && export MAKE_ARGS="-j4"

SHOW_IMG="mctest-show:test-latest"
TMP_FOLDER="$SCRIPT_DIR/../../.tmp"
TMP_PREBASE_FOLDER="$TMP_FOLDER/tests/pre/base"

GODOG_CONCURRENCY="${GODOG_CONCURRENCY:-1}"
{ [ -n "$GODOG_WIP" ] && GODOG_TAGS="--godog.tags=wip"; } || \
    GODOG_TAGS="${GODOG_TAGS:---godog.tags=~disabled}"
GODOG_ARGS="--godog.concurrency $GODOG_CONCURRENCY"
GODOG_EXTRA_ARGS="${GODOG_EXTRA_ARGS:-}"

prepare_prebase_folder()
{
    # create base folders
    mkdir -p "$TMP_FOLDER" "$TMP_PREBASE_FOLDER/config/default" || return 1

    # copy code to temp folder
    { rsync \
        --info=progress2 \
        --recursive \
        --chmod=0755 \
        --chown="$(id -u):$(id -g)" \
        demo \
        "$TMP_PREBASE_FOLDER" && \
            $MAKE --directory "$TMP_PREBASE_FOLDER/demo/show" kustomize manifests generate; } || return 1

    # build default manifests
    ( cd "$TMP_PREBASE_FOLDER/demo/show/config" && \
        ( cd "manager" && \
            ../../bin/kustomize edit set image controller="$SHOW_IMG" && \
            ../../bin/kustomize build . > "$TMP_PREBASE_FOLDER/config/default/show.yaml" \
        ) && ( cd "rbac" && \
            ../../bin/kustomize build . > "$TMP_PREBASE_FOLDER/config/default/show-rbac.yaml" ) )
}

build_images()
{
    $MAKE --directory="$TMP_PREBASE_FOLDER/demo/show" IMG="$SHOW_IMG" docker-build
}

init()
{
    # prepare, test base folder and build images
    print_section "preparing manifests and images"
    rm -rf "$TMP_FOLDER" || true
    { \
        $MAKE --directory demo/show kustomize generate manifests && \
            prepare_prebase_folder && \
            build_images; \
    } || { print_error "error initializing preparing base folder and/or building images"; exit 1; }
}

reload_images()
{
    print_section "reloading mctest images into management cluster"
    $KIND load docker-image --name "$CLUSTER_NAME" "$SHOW_IMG" || \
        { print_error "error loading mctest's docker images into management cluster"; exit 1; }
}

run_tests()
{
    go -C "$SCRIPT_DIR/.." \
        test \
            -v \
            "$GODOG_TAGS" \
            "$GODOG_ARGS" \
            "$GODOG_EXTRA_ARGS"
}

