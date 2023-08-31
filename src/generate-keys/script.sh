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
# The certificate will be issued for the Common Name (CN) of `webhook-server.$DATASET_OPERATOR_NAMESPACE.svc`, which is the
# cluster-internal DNS name for the service.
#
# NOTE: THIS SCRIPT EXISTS FOR DEMO PURPOSES ONLY. DO NOT USE IT FOR YOUR PRODUCTION WORKLOADS.
# Generate the CA cert and private key

# Ensure DATASET_OPERATOR_NAMESPACE is defined
DATASET_OPERATOR_NAMESPACE="${DATASET_OPERATOR_NAMESPACE:-dlf}"

# Perform the operations in a temporary directory
mkdir -p /tmp/dlf-keys

# Generate the Admission Controller Webhook CA key and cert
openssl req -nodes -new -x509 -keyout /tmp/dlf-keys/ca.key -out /tmp/dlf-keys/ca.crt -subj "/CN=Admission Controller Webhook CA" -days 10000

# Generate the private key for the webhook server
openssl genrsa -out /tmp/dlf-keys/webhook-server-tls.key 2048

# Generate a Certificate Signing Request (CSR) for the private key, and sign it with the private key of the CA.
cat >/tmp/dlf-keys/csr.conf <<EOF
[req]
default_bits = 2048
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn
[dn]
CN = webhook-server.$DATASET_OPERATOR_NAMESPACE.svc
[req_ext]
subjectAltName = @alt_names
[alt_names]
DNS.1 = webhook-server.$DATASET_OPERATOR_NAMESPACE.svc
[v3_ext]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=serverAuth,clientAuth
subjectAltName=@alt_names
EOF
openssl req -new -key /tmp/dlf-keys/webhook-server-tls.key -config /tmp/dlf-keys/csr.conf | openssl x509 -req -CA /tmp/dlf-keys/ca.crt -CAkey /tmp/dlf-keys/ca.key -set_serial 01 -days 10000 -extensions v3_ext -extfile /tmp/dlf-keys/csr.conf -out /tmp/dlf-keys/webhook-server-tls.crt
rm /tmp/dlf-keys/csr.conf

# AP - exporting these variables is essential for the 
# envsubst call to work
export CA_PEM_B64="$(openssl base64 -A < "/tmp/dlf-keys/ca.crt")"
export DATASET_OPERATOR_NAMESPACE="${DATASET_OPERATOR_NAMESPACE:-dlf}"

kubectl -n $DATASET_OPERATOR_NAMESPACE create secret tls webhook-server-tls \
            --cert "/tmp/dlf-keys/webhook-server-tls.crt" \
            --key "/tmp/dlf-keys/webhook-server-tls.key" --dry-run -o yaml | kubectl apply -f -
kubectl -n $DATASET_OPERATOR_NAMESPACE label secret/webhook-server-tls app.kubernetes.io/name=dlf
envsubst < "webhook.yaml.template" | kubectl apply -n $DATASET_OPERATOR_NAMESPACE -f -

rm -rf /tmp/dlf-keys