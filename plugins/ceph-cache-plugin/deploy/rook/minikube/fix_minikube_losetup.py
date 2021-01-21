#!/bin/python3
import subprocess
import json


def run_cmd(cmd):
    proc = subprocess.Popen([cmd], stdout=subprocess.PIPE, stderr=subprocess.PIPE, shell=True, bufsize=0, universal_newlines=True)
    (out, err) = proc.communicate()
    return out.strip(), err.strip()


def check_busybox_losetup(node_name, node_ip):
    cmd = f"ssh -o LogLevel=ERROR -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i \"${{" \
          f"HOME}}/.minikube/machines/{node_name}/id_rsa\" docker@{node_ip} 'losetup --version' "
    out, err = run_cmd(cmd)
    return "BusyBox" in err or "Busybox" in out


def fix_busybox_losetup(node_name, node_ip):
    cmd = f"scp -o LogLevel=ERROR -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i \"${{" \
          f"HOME}}/.minikube/machines/{node_name}/id_rsa\" /sbin/losetup docker@{node_ip}:~/losetup "
    out, err = run_cmd(cmd)
    if err != "":
        return False
    cmd = f"ssh -o LogLevel=ERROR -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i \"${{" \
          f"HOME}}/.minikube/machines/{node_name}/id_rsa\" docker@{node_ip} 'sudo sh -c \"rm -f /sbin/losetup && cp " \
          f"~docker/losetup /sbin\"' "
    out, err = run_cmd(cmd)
    if err != "":
        return False
    return not check_busybox_losetup(node_name, node_ip)


out, err = run_cmd("minikube profile list --output=json")
minikubeConfig = json.loads(out)
for cluster in minikubeConfig["valid"]:
    if cluster["Status"] != "Running":
        continue
    cluster_name = cluster["Name"]

    for idx, node in enumerate(cluster["Config"]["Nodes"]):
        node_name = cluster_name
        if node["Name"] != "":
            node_name += "-" + node["Name"]
        node_ip = node["IP"]

        if check_busybox_losetup(node_name, node_ip):
            print(f"fixing {node_name} losetup")
            if fix_busybox_losetup(node_name, node_ip):
                print(f"{node_name} losetup fixed")
            else:
                print(f"{node_name} losetup failed")
        else:
            print(f"{node_name} loset already fixed")