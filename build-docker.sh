#!/bin/bash

# builds the docker image, and optionally pushes to the ghcr registry.
DOCKER_BIN=$(which podman docker | grep -v found | head -n 1)

SHOULD_PUSH=0
IMAGE_NAME="ghcr.io/{{project.vendor.github}}/{{project.name}}"
IMAGE_TAG="latest"
DOCKERFILENAME="./Dockerfile"
VARIANT="base"
DEBUGMODE=0
DRY_RUN=0
REGISTRY_LOGIN=0
DOCKER_LOGIN_ARG=""
DOCKER_PW_ARG=""
REGISTRY_DOMAIN="ghcr.io"
DOCKER_PLATFORM_TARGET="linux/amd64"

if [ -z "$DOCKER_BIN" ]; then
    DOCKER_BIN="docker"
fi

for arg in "$1" "$2" "$3" "$4" "$5"; do
  case "$arg" in
    --docker) DOCKER_BIN=$(which docker) ;;
    --podman) DOCKER_BIN=$(which podman) ;;
    --login) DOCKER_LOGIN_ARG="$arg" ;;
    --password) DOCKER_PW_ARG="$arg" ;;
    --registry) REGISTRY_DOMAIN="$arg" ;;
    --push) SHOULD_PUSH=1 ;;
    --debug) DEBUGMODE=1 ;;
    --dry-run) DRY_RUN=1 ;;
  esac
done

IMAGE_VERSION=$(git describe --tags --always --abbrev=10)

if [ "$DOCKER_LOGIN_ARG" != "" ] && [ "$DOCKER_PW_ARG" != "" ]; then
    $DOCKER_BIN login $REGISTRY_DOMAIN -u "$DOCKER_LOGIN_ARG" --password "$DOCKER_PW_ARG"
else
    if [ $SHOULD_PUSH -eq 1 ]; then
        echo "ERROR: --login and --password are required when using --push"
        echo "NOTE: You can use a GitHub personal access token as password for ghcr.io"
        exit 1
    fi
fi

if [ "$VARIANT" == "base" ]; then
    IMAGE_TAG="latest"
    DOCKERFILENAME="./Dockerfile"
fi

if [ $DEBUGMODE -eq 1 ]; then
    echo "[debug] using $DOCKER_BIN"
    echo "[debug] build image $IMAGE_NAME:$IMAGE_TAG using $DOCKERFILENAME"

    if [ $SHOULD_PUSH -eq 1 ]; then
        echo "[debug] push image to dockerhub"
    else
        echo "[debug] do not push image to dockerhub"
    fi
fi

if [ $DRY_RUN -eq 1 ]; then
    echo "[dry-run] " "$DOCKER_BIN" build --platform="$DOCKER_PLATFORM_TARGET" -t "$IMAGE_NAME:$IMAGE_VERSION" -f $DOCKERFILENAME .

    if [ $SHOULD_PUSH -eq 1 ]; then
        echo "[dry-run] " "$DOCKER_BIN" push $IMAGE_NAME:$IMAGE_VERSION
        echo "[dry-run] " "$DOCKER_BIN" push $IMAGE_NAME:$IMAGE_TAG
    fi

    exit 0
fi

$DOCKER_BIN build --platform="$DOCKER_PLATFORM_TARGET" -t "$IMAGE_NAME:$IMAGE_VERSION" -f $DOCKERFILENAME .
$DOCKER_BIN tag $IMAGE_NAME:$IMAGE_VERSION $IMAGE_NAME:$IMAGE_TAG

if [ $SHOULD_PUSH -eq 1 ]; then
    $DOCKER_BIN push $IMAGE_NAME:$IMAGE_VERSION
    $DOCKER_BIN push $IMAGE_NAME:$IMAGE_TAG
fi
