#!/bin/bash


DATASET_OPERATOR_NAMESPACE="${DATASET_OPERATOR_NAMESPACE:-default}"
DOCKER_REGISTRY_COMPONENTS="${DOCKER_REGISTRY_COMPONENTS:-the_registry_to_use_for_components}"
HIVESERVER_IMAGE="hive-server:latest"

function check_env(){
    echo "Checking if S3 connection variables are available"
    if [[ -z "$S3_ENDPOINT" ]]; then
       echo "Using Nooba for connection credentials"	
       if [[ -z "$NOOBAA_HOME" ]]; then
          echo "Noobaa install cannot be found"
          exit 1
       fi
       export S3_ENDPOINT=$(minikube service s3 --url | head -n1)
       export AWS_ACCESS_KEY_ID=$(${NOOBAA_HOME}/noobaa status 2>/dev/null | grep AWS_ACCESS_KEY_ID | awk -F ": " '{print $2}')
       export AWS_SECRET_ACCESS_KEY=$(${NOOBAA_HOME}/noobaa status 2>/dev/null | grep AWS_SECRET_ACCESS_KEY | awk -F ": " '{print $2}')
    fi
}

function populate_hive(){
    echo "Populating Hive with the book table"
    HIVE_CLI_PORT=`kubectl get svc hiveserver -n ${DATASET_OPERATOR_NAMESPACE} -o jsonpath='{.spec.ports[?(@.name=="cliservice")].nodePort}'`
    HIVE_CLI_IP=`minikube service hiveserver --url -n ${DATASET_OPERATOR_NAMESPACE} | awk -F':' -v port="$HIVE_CLI_PORT" '{if ($3 == port) print $2}' - | cut -d / -f 3`
    docker run -v ${PWD}:/sampleapp -it --network host ${DOCKER_REGISTRY_COMPONENTS}/${HIVESERVER_IMAGE} bin/beeline -u "jdbc:hive2://$HIVE_CLI_IP:$HIVE_CLI_PORT/;transportMode=http;httpPath=/cliservice" -f /sampleapp/sample.hql
    if [ $? -eq 0 ]
    then
        echo "Hive successfully populated"
    fi

}


function build_awscli_image(){
    echo "Building image for S3 commands"
    docker build -f ${NOOBAA_HOME}/Dockerfile-awscli-alpine -t awscli-alpine . > /dev/null 2>&1

    if [ $? -eq 0 ]
    then
        echo "AWS image successfully built"
    fi
}

function create_s3_dataset(){
    echo "Creating S3 bucket and uploading data"
    docker run --rm --network host \
           -e AWS_ACCESS_KEY_ID \
               -e AWS_SECRET_ACCESS_KEY \
               awscli-alpine \
               aws --endpoint ${S3_ENDPOINT} \
               s3 mb s3://book-test

    if [ $? -eq 0 ]
    then
        echo "Bucket book-test successfully created"
    fi

    docker run --rm --network host \
           -e AWS_ACCESS_KEY_ID \
               -e AWS_SECRET_ACCESS_KEY \
               -v  ${PWD}:/sampleapp \
               awscli-alpine \
               aws --endpoint ${S3_ENDPOINT} \
               s3 cp /sampleapp/books.csv s3://book-test/

    if [ $? -eq 0 ]
    then
        echo "books.csv successfully uploaded"
    fi
}

function create_book_dataset(){
    echo "Creating the Book dataset object"
    envsubst < bookdataset.yaml | kubectl apply -f -
    kubectl apply -f samplepod.yaml
    kubectl wait --for=condition=ready pods --all > /dev/null 2>&1
}

function test_book_dataset(){
    echo "Checking if the Book dataset is available"
    kubectl exec -it sampleapp cat /mnt/datasets/bookds/books.csv
}

check_env
build_awscli_image
create_s3_dataset
populate_hive
create_book_dataset
test_book_dataset
