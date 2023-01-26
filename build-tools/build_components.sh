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

if [ $BUILD_AND_PUSH = "yes" ]; then
      if [ $SKIP_LOGIN = "no" ]; then
            echo $REGISTRY_PASSWORD | docker login -u $REGISTRY_USERNAME --password-stdin $REGISTRY_URL
      fi
      if [ $CREATE_NEW_BUILDX_CONTEXT = "yes" ]; then
            docker buildx create --use
      fi
      (cd ../src/dataset-operator && ./build_and_push_multiarch_dataset_operator.sh)
      (cd ../src/generate-keys && ./build_and_push_multiarch_generate_keys.sh)
else
      (cd ../src/dataset-operator && ./build_dataset_operator.sh)
      (cd ../src/generate-keys && ./build_generate_keys.sh)
fi
