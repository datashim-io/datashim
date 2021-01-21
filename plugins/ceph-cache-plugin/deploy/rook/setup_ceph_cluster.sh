#!/bin/bash

# For Minikube installation see the tips here ./minikube/

# kubectl addon to execute root commands on the each node
# https://github.com/kvaps/kubectl-node-shell
KUBENODESHELL=https://github.com/kvaps/kubectl-node-shell/raw/master/kubectl-node_shell

# remove rook installation
remove_rook_installation () {
  # check if rook-ceph is installed
  if [[ $(kubectl get namespaces/rook-ceph --ignore-not-found | wc -l) -eq 0 ]]; then
    echo "Rook is not installed"
    return
  fi

  # first detach the finalizers
  kubectl patch crd/cephclusters.ceph.rook.io -p '{"metadata":{"finalizers":[]}}' --type=merge > /dev/null
  kubectl patch crd/cephobjectstores.ceph.rook.io -p '{"metadata":{"finalizers":[]}}' --type=merge > /dev/null
  kubectl patch cephclusters.ceph.rook.io/rook-ceph -p '{"metadata":{"finalizers":[]}}' --type=merge -n rook-ceph > /dev/null
  kubectl patch crd/cephobjectstoreusers.ceph.rook.io -p '{"metadata":{"finalizers":[]}}' --type=merge > /dev/null
  kubectl patch crd/cephblockpools.ceph.rook.io -p '{"metadata":{"finalizers":[]}}' --type=merge > /dev/null
  kubectl patch crd/cephclients.ceph.rook.io -p '{"metadata":{"finalizers":[]}}' --type=merge > /dev/null
  kubectl patch crd/cephfilesystems.ceph.rook.io -p '{"metadata":{"finalizers":[]}}' --type=merge > /dev/null
  kubectl patch crd/cephnfses.ceph.rook.io -p '{"metadata":{"finalizers":[]}}' --type=merge > /dev/null
  kubectl patch crd/cephobjectrealms.ceph.rook.io  -p '{"metadata":{"finalizers":[]}}' --type=merge > /dev/null
  kubectl patch crd/cephobjectzonegroups.ceph.rook.io  -p '{"metadata":{"finalizers":[]}}' --type=merge > /dev/null
  kubectl patch crd/cephobjectzones.ceph.rook.io  -p '{"metadata":{"finalizers":[]}}' --type=merge > /dev/null
  kubectl patch crd/cephrbdmirrors.ceph.rook.io -p '{"metadata":{"finalizers":[]}}' --type=merge > /dev/null

  # delete the rook-cluster
  kubectl delete cephcluster/rook-ceph -n rook-ceph > /dev/null
  kubectl delete -f ./common.yaml > /dev/null

  # wait for all associated pods with rook to terminate
  while [[ $(kubectl get pods -n rook-ceph --ignore-not-found | wc -l ) -ne 0 ]]; do
    echo "Waiting for Rook pods to terminate"
    sleep 2
  done

  # clean rook leftovers
  for node in $(kubectl get nodes --no-headers | awk '{print $1}') ; do
    PATH="${PATH}:${PWD}" kubectl node_shell "$node" -- rm -rf /var/lib/rook > /dev/null
  done

  echo "Rook uninstalled"
}

# remove csi lvm installation
remove_csi_lvm () {
  # check if csi-lvm is installed
  if [[ $(kubectl get namespaces/csi-lvm --ignore-not-found | wc -l) -eq 0 ]]; then
    echo "CSI lvm is not installed"
    return
  fi

  # remove csi lvm yaml and namespace
  kubectl delete -f ./csi-lvm-setup/reviver.yaml > /dev/null
  kubectl delete -f ./csi-lvm-setup/controller.yaml > /dev/null
  kubectl delete -f ./csi-lvm-setup/create-loops.yaml > /dev/null
  kubectl delete namespace/csi-lvm > /dev/null

  # wait for all the pods of csi-lvm to terminate
  while [[ $(kubectl get pods -n csi-lvm --ignore-not-found | wc -l ) -ne 0 ]]; do
    echo "Waiting for CSI lvm pods to terminate"
    sleep 2
  done

  # unlabel nodes and clean the csi-lvm leftovers
  for node in $(kubectl get nodes --selector=use-csi-lvm=true --no-headers | awk '{print $1}') ; do
    kubectl label node/"$node" use-csi-lvm- > /dev/null
    PATH="${PATH}:${PWD}" kubectl node_shell "$node" -- losetup -d /dev/loop111
    PATH="${PATH}:${PWD}" kubectl node_shell "$node" -- rm -rf /etc/lvm/cache/.cache > /dev/null
  done

  echo "CSI lvm uninstalled"
}

# download node-shell kubectl plugin
# https://github.com/kvaps/kubectl-node-shell
download_node_shell () {
  if [[ ! -f "kubectl-node_shell" ]]; then
    wget $KUBENODESHELL --output-document=kubectl-node_shell
    if [[ $? -ne 0 ]]; then
      echo "Failed to download kubectl-node_shell"
      exit
    fi
    chmod +x ./kubectl-node_shell
  fi
}

# install csi lvm
install_csi_lvm_rook () {
  # label appropriately the nodes specified by the user
  for node in $(kubectl get nodes --no-headers | awk '{print $1}') ; do
    for item in "$@"; do
      if [[ "$node" == "$item" ]]; then
        kubectl label node/"$node" use-csi-lvm=true --overwrite > /dev/null
      fi
    done
  done

  # check that at least one node has the label use-csi-lvm=true
  if [[ $(kubectl get nodes --selector=use-csi-lvm=true --ignore-not-found | wc -l) -eq 0 ]]; then
    echo "No nodes with use-csi-lvm label found"
    exit
  fi

  # install csi-lvm yamls
  kubectl create namespace csi-lvm --dry-run=client -o yaml | kubectl apply -f -
  kubectl create -f ./csi-lvm-setup/create-loops.yaml > /dev/null
  kubectl create -f  ./csi-lvm-setup/controller.yaml > /dev/null
  kubectl create -f  ./csi-lvm-setup/reviver.yaml > /dev/null

  # wait for all the pods to run
  kubectl wait --for=condition=ready pod --all -n csi-lvm > /dev/null

  install_rook_cluster
}

# install rook cluster
install_rook_cluster () {
  #install yamls
  kubectl create -f common.yaml > /dev/null
  kubectl create -f operator.yaml > /dev/null
  kubectl create -f cluster-on-pvc.yaml > /dev/null
}

download_node_shell

remove_rook_installation

# unlabel nodes and clean the csi-lvm leftovers
#for node in $(kubectl get nodes --selector=use-csi-lvm=true --no-headers | awk '{print $1}') ; do
#  kubectl label node/"$node" use-csi-lvm- > /dev/null
#done
#
#for node in $(kubectl get nodes --no-headers | awk '{print $1}') ; do
#  for item in "$@"; do
#    if [[ "$node" == "$item" ]]; then
#      kubectl label node/"$node" use-csi-lvm=true --overwrite > /dev/null
#    fi
#  done
#done

#install_rook_cluster
remove_csi_lvm

install_csi_lvm_rook "$@"
