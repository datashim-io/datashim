#!/bin/bash

echo "Make sure you replace YOUR_REGISTRY,YOUR_EMAIL,YOUR_PASSWORD with the correct values for your docker registry"

kubectl create secret docker-registry docker-secret -n rook-ceph --docker-server=https://YOUR_REGISTRY/v2/  --docker-username=YOUR_EMAIL --docker-email=YOUR_EMAIL --docker-password=YOUR_PASSWORD
kubectl patch serviceaccount default -p '{"imagePullSecrets": [{"name": "docker-secret"}]}' -n rook-ceph
kubectl patch serviceaccount rook-ceph-cmd-reporter -p '{"imagePullSecrets": [{"name": "docker-secret"}]}' -n rook-ceph
kubectl patch serviceaccount rook-ceph-mgr -p '{"imagePullSecrets": [{"name": "docker-secret"}]}' -n rook-ceph
kubectl patch serviceaccount rook-ceph-osd -p '{"imagePullSecrets": [{"name": "docker-secret"}]}' -n rook-ceph
kubectl patch serviceaccount rook-ceph-system -p '{"imagePullSecrets": [{"name": "docker-secret"}]}' -n rook-ceph
kubectl patch serviceaccount rook-csi-cephfs-plugin-sa -p '{"imagePullSecrets": [{"name": "docker-secret"}]}' -n rook-ceph
kubectl patch serviceaccount rook-csi-cephfs-provisioner-sa -p '{"imagePullSecrets": [{"name": "docker-secret"}]}' -n rook-ceph
kubectl patch serviceaccount rook-csi-rbd-plugin-sa -p '{"imagePullSecrets": [{"name": "docker-secret"}]}' -n rook-ceph
kubectl patch serviceaccount rook-csi-rbd-provisioner-sa -p '{"imagePullSecrets": [{"name": "docker-secret"}]}' -n rook-ceph

