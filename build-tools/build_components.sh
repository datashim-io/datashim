#!/bin/bash

print_usage() {
    echo "Usage: $0 [-p]"
    echo "Use -p to build and push multiarch images"
}

BUILD_AND_PUSH="no"
while getopts 'p' OPTION
do
    case "$OPTION" in
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
      cd ../src/dataset-operator && ./build_and_push_multiarch_dataset_operator.sh
      cd ../src/generate-keys && ./build_and_push_multiarch_generate_keys.sh
else
      cd ../src/dataset-operator && ./build_dataset_operator.sh
      cd ../src/generate-keys && ./build_generate_keys.sh
fi
