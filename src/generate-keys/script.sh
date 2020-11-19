#!/usr/bin/env bash

# Copyright (c) 2019 StackRox Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# generate-keys.sh
#
# Generate a (self-signed) CA certificate and a certificate and private key to be used by the webhook demo server.
# The certificate will be issued for the Common Name (CN) of `webhook-server.webhook-demo.svc`, which is the
# cluster-internal DNS name for the service.
#
# NOTE: THIS SCRIPT EXISTS FOR DEMO PURPOSES ONLY. DO NOT USE IT FOR YOUR PRODUCTION WORKLOADS.
# Generate the CA cert and private key

mkdir -p /tmp/dlf-keys

openssl req -nodes -new -x509 -keyout /tmp/dlf-keys/ca.key -out /tmp/dlf-keys/ca.crt -subj "/CN=Admission Controller Webhook CA"
# Generate the private key for the webhook server
openssl genrsa -out /tmp/dlf-keys/webhook-server-tls.key 2048
# Generate a Certificate Signing Request (CSR) for the private key, and sign it with the private key of the CA.
openssl req -new -key /tmp/dlf-keys/webhook-server-tls.key -subj "/CN=webhook-server.$DATASET_OPERATOR_NAMESPACE.svc" \
    | openssl x509 -req -CA /tmp/dlf-keys/ca.crt -CAkey /tmp/dlf-keys/ca.key -CAcreateserial -out /tmp/dlf-keys/webhook-server-tls.crt

export CA_PEM_B64="$(openssl base64 -A < "/tmp/dlf-keys/ca.crt")"

export DATASET_OPERATOR_NAMESPACE="${DATASET_OPERATOR_NAMESPACE:-dlf}"

kubectl -n $DATASET_OPERATOR_NAMESPACE create secret tls webhook-server-tls \
            --cert "/tmp/dlf-keys/webhook-server-tls.crt" \
            --key "/tmp/dlf-keys/webhook-server-tls.key" --dry-run -o yaml | kubectl apply -f -
kubectl -n $DATASET_OPERATOR_NAMESPACE label secret/webhook-server-tls app.kubernetes.io/name=dlf
envsubst < "webhook.yaml.template" | kubectl apply -n $DATASET_OPERATOR_NAMESPACE -f -

rm -rf /tmp/dlf-keys