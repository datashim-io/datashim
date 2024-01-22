#!/bin/bash
set -e 

print_usage() {
    echo "usage: $0 [-p] <REGISTRY_URL> <VERSION>"
    echo "Use -p to build and push multiarch images"
}

PUSH="false"
while getopts 'p' flag; do
case "$flag" in
    p) 
        PUSH="true"
        ;;
    ?) 
        print_usage >&2
        exit 1
        ;;
esac
done

shift $((OPTIND-1))

REGISTRY_URL="${1:-quay.io/datashim-io}"
VERSION="${2:-latest}"

docker_build () {
    docker buildx build --platform linux/amd64 -t ${REGISTRY_URL}/generate-keys:${VERSION} .
    docker buildx build --load -t ${REGISTRY_URL}/generate-keys:${VERSION} .
}

docker_build_and_push () {
    docker buildx build --platform linux/amd64,linux/arm64 --push -t ${REGISTRY_URL}/generate-keys:${VERSION} .
}

podman_build () {
    podman manifest create ${REGISTRY_URL}/generate-keys:${VERSION}
    podman buildx build --platform linux/amd64,linux/arm64  --manifest ${REGISTRY_URL}/generate-keys:${VERSION} .
}

podman_push () {
    podman manifest push ${REGISTRY_URL}/generate-keys:${VERSION} 
 
}

DOCKERCMD="docker"
ALTDOCKERCMD="podman"
if !(command -v ${DOCKERCMD} &> /dev/null)
then
    echo "Docker command not found"
    if !(command -v ${ALTDOCKERCMD} &> /dev/null)
    then
        echo "Neither ${DOCKERCMD} nor ${ALTDOCKERCMD} commands found.. cannot build "
        exit 1
    else
        DOCKERCMD=${ALTDOCKERCMD}  
    fi
else
    echo "Docker command found"
    cmd_type=$(type -t ${DOCKERCMD})
    if [ $cmd_type == "alias" ]
    then
        echo "${DOCKERCMD} is an alias, switching to ${ALTDOCKERCMD}"
        DOCKERCMD=${ALTDOCKERCMD}  
    fi
fi 

if [ $PUSH == "true" ]
then
    echo "pushing images to the registry"
    if [ ${DOCKERCMD} == "docker" ]
    then
        docker_build_and_push
    else 
        podman_build
        podman_push
    fi
else
    echo "building image locally"
    if [ ${DOCKERCMD} == "docker" ]
    then
        docker_build 
    else
        podman_build
    fi
fi
