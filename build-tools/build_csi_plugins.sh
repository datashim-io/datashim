#!/bin/bash
set -e

print_usage() {
    echo "Usage: $0 [-k] [-p] [-s]"
    echo "Use -k to avoid creating another buildx context"
    echo "Use -p to build and push multiarch images"
    echo "Use -s to skip logging in to the container registry"
}

BUILD_AND_PUSH="no"
CREATE_NEW_BUILDX_CONTEXT="yes"
SKIP_LOGIN="no"
while getopts 'kps' OPTION
do
    case "$OPTION" in
        k)
            CREATE_NEW_BUILDX_CONTEXT="no"
            ;;
        p)
            BUILD_AND_PUSH="yes"
            ;;
        s)
            SKIP_LOGIN="yes"
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

if [ ${DOCKERCMD} == "docker" ]
then 
    if [ $CREATE_NEW_BUILDX_CONTEXT = "yes" ]; then
        docker buildx create --use
    fi
fi

if [ $BUILD_AND_PUSH = "yes" ]; then
      if [ $SKIP_LOGIN = "no" ]; then
            echo $REGISTRY_PASSWORD | docker login -u $REGISTRY_USERNAME --password-stdin $REGISTRY_URL
      fi
      (cd ../src/csi-s3 && ./build_multiarch_csis3.sh -p $REGISTRY_URL $VERSION)
      (cd ../src/csi-nfs && ./build_multiarch_csinfs.sh -p $REGISTRY_URL $VERSION)
else
      (cd ../src/csi-s3 && ./build_multiarch_csis3.sh $REGISTRY_URL $VERSION)
      (cd ../src/csi-nfs && ./build_multiarch_csinfs.sh $REGISTRY_URL $VERSION)
fi
