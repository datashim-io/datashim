#!/bin/bash 

DATASET_OPERATOR_NAMESPACE=default

HIVE_WEB_PORT=`kubectl get svc hiveserver -n ${DATASET_OPERATOR_NAMESPACE} -o jsonpath='{.spec.ports[?(@.name=="web-ui")].nodePort}'`
HIVE_WEB_IP=`minikube service hiveserver --url -n ${DATASET_OPERATOR_NAMESPACE} | awk -F':' -v port="$HIVE_WEB_PORT" '{if ($3 == port) print $2}' - | cut -d / -f 3`

echo "You can open http://${HIVE_WEB_IP}:${HIVE_WEB_PORT} in a browser.."
echo "Testing using curl"
curl -sSIf http://${HIVE_WEB_IP}:${HIVE_WEB_PORT} | head -n1
