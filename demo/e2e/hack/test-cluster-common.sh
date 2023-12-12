#!/bin/bash

[ -z "$CLUSTERCTL" ] && export CLUSTERCTL="clusterctl"
[ -z "$KIND" ] && export KIND="kind"
[ -z "$KIND_CONFIG" ] && export KIND_CONFIG="$SCRIPT_DIR/../config/management-cluster/kind-cluster-with-extramounts.yaml"
[ -z "$CLUSTER_NAME" ] && export CLUSTER_NAME="mctest-e2e-mgmt"
[ -z "$KUBECTL" ] && export KUBECTL="kubectl"
[ -z "$TIMEOUT" ] && export TIMEOUT="120s"
[ -z "$MAKE" ] && export MAKE="make"
[ -z "$MAKE_ARGS" ] && export MAKE_ARGS="-j4"

SHOW_IMG="mctest-show:test-latest"
TMP_FOLDER="$SCRIPT_DIR/../../.tmp"
TMP_PREBASE_FOLDER="$TMP_FOLDER/tests/pre"

prepare_prebase_folder()
{
    chmod -R 0755 "$TMP_FOLDER" || true
    rm -rf "$TMP_FOLDER" || true

    # create base folders
    mkdir -p "$TMP_FOLDER" "$TMP_PREBASE_FOLDER/config/default" || return 1

    # copy code to temp folder
    { mkdir "$TMP_PREBASE_FOLDER/demo" && rsync  \
        --info=progress2 \
        --recursive \
        --chmod=0755 \
        --chown="$(id -u):$(id -g)" \
        demo/e2e demo/show \
        "$TMP_PREBASE_FOLDER/demo"; } || return 1

    # build default manifests
    ( cd "$TMP_PREBASE_FOLDER/demo/show/config" && \
        ( cd "manager" && \
            ../../bin/kustomize edit set image controller="$SHOW_IMG" && \
            ../../bin/kustomize build . > "$TMP_PREBASE_FOLDER/config/default/show.yaml" \
        ) && ( cd "rbac" && \
            ../../bin/kustomize build . > "$TMP_PREBASE_FOLDER/config/default/show-rbac.yaml" ) )

    chmod -R 0755 "$TMP_FOLDER" || true
}

build_images()
{
    $MAKE --directory="$TMP_PREBASE_FOLDER/demo/show" IMG="$SHOW_IMG" docker-build
}

init()
{
    # prepare, test base folder and build images
    print_section "preparing manifests and images"
    { \
        $MAKE --directory demo/show kustomize generate manifests && \
            prepare_prebase_folder && \
            build_images; \
    } || { \
        print_error "error initializing preparing base folder and/or building images"; \
        chmod -R 0755 "$TMP_FOLDER" || true ; \
        exit 1; \
    }
}

reload_images()
{
    print_section "reloading mctest images into management cluster"
    $KIND load docker-image --name "$CLUSTER_NAME" "$SHOW_IMG" || \
        { print_error "error loading mctest's docker images into management cluster"; exit 1; }
}
