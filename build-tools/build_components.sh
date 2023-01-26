#!/bin/bash
set -e

print_usage() {
    echo "Usage: $0 [-p] [-k]"
    echo "Use -k to avoid creating another buildx context"
    echo "Use -p to build and push multiarch images"
}

BUILD_AND_PUSH="no"
CREATE_NEW_BUILDX_CONTEXT='yes'
while getopts 'pk' OPTION
do
    case "$OPTION" in
        k)
            CREATE_NEW_BUILDX_CONTEXT="no"
            ;;
        p)
            BUILD_AND_PUSH="yes"
            ;;
        ?)
            print_usage >&2
            exit 1
            ;;
    esac
done

if [ $BUILD_AND_PUSH = "yes" ]; then
      docker login -u $REGISTRY_USERNAME -p $REGISTRY_PASSWORD $REGISTRY_URL
      if [ $CREATE_NEW_BUILDX_CONTEXT = "yes" ]; then
            docker buildx create --use
      fi
      (cd ../src/dataset-operator && ./build_and_push_multiarch_dataset_operator.sh)
      (cd ../src/generate-keys && ./build_and_push_multiarch_generate_keys.sh)
else
      (cd ../src/dataset-operator && ./build_dataset_operator.sh)
      (cd ../src/generate-keys && ./build_generate_keys.sh)
fi
