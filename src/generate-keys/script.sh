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
openssl req -nodes -new -x509 -keyout /tmp/ca.key -out /tmp/ca.crt -subj "/CN=Admission Controller Webhook CA"
# Generate the private key for the webhook server
openssl genrsa -out /tmp/webhook-server-tls.key 2048
# Generate a Certificate Signing Request (CSR) for the private key, and sign it with the private key of the CA.
openssl req -new -key /tmp/webhook-server-tls.key -subj "/CN=webhook-server.$DATASET_OPERATOR_NAMESPACE.svc" \
    | openssl x509 -req -CA /tmp/ca.crt -CAkey /tmp/ca.key -CAcreateserial -out /tmp/webhook-server-tls.crt

export CA_PEM_B64="$(openssl base64 -A < "/tmp/ca.crt")"

echo $DATASET_OPERATOR_NAMESPACE
echo $CA_PEM_B64

/tmp/kubectl -n $DATASET_OPERATOR_NAMESPACE create secret tls webhook-server-tls \
            --cert "/tmp/webhook-server-tls.crt" \
            --key "/tmp/webhook-server-tls.key" --dry-run -o yaml | /tmp/kubectl apply -f -
envsubst < "/tmp/webhook.yaml.template" | /tmp/kubectl apply -n $DATASET_OPERATOR_NAMESPACE -f -

cd ~
rm -rf /tmp/*.crt
rm -rf /tmp/*.key