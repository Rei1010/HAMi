#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -x


REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${REPO_ROOT}"

source "${REPO_ROOT}"/hack/util.sh

function install_govc() {
    local govc_version="v0.37.3"
    local govc_tar_name="govc_Linux_x86_64.tar.gz"
    local govc_tar_url="https://github.com/vmware/govmomi/releases/download/${govc_version}/${govc_tar_name}"

    wget -q $govc_tar_url || { echo "Failed to download govc"; exit 1; }
    tar -zxvf ${govc_tar_name}
    mv govc /usr/local/bin/
    govc version
}

function govc_poweron_vm() {
    local vm_name=${1:-""}
    if [[ -z "$vm_name" ]]; then
        echo "Error: VM name is required"
        return 1
    fi

    govc vm.power -on "$vm_name"
    echo -e "\033[35m === $vm_name: power turned on === \033[0m"
    while true; do
        if [[ $(govc vm.info "$vm_name" | grep -c poweredOn) -eq 1 ]]; then
            break
        fi
        sleep 5
    done
}

function govc_poweroff_vm() {
    local vm_name=${1:-""}
    if [[ -z "$vm_name" ]]; then
        echo "Error: VM name is required"
        return 1
    fi

    if [[ $(govc vm.info "$vm_name" | grep -c poweredOn) -eq 1 ]]; then
        govc vm.power -off -force "$vm_name"
        echo -e "\033[35m === $vm_name has been down === \033[0m"
    fi
}

function govc_restore_vm_snapshot() {
    local vm_name=${1:-""}
    local vm_snapshot_name=${2:-""}

    govc snapshot.revert -vm "$vm_name" "$vm_snapshot_name"
    echo -e "\033[35m === $vm_name reverted to snapshot: $(govc snapshot.tree -vm "$vm_name" -C -D -i -d) === \033[0m"
}

function setup_gpu_test_env() {

    # env variables come from SECRET
    export GOVC_USERNAME=$VSPHERE_USER
    export GOVC_PASSWORD=$VSPHERE_PASSWD
    export GOVC_URL=$VSPHERE_SERVER
    export GOVC_DATACENTER=$VSPHERE_DATACENTER
    export vm_name=$VSPHERE_GPU_VM_NAME
    export vm_snapshot_name=$VSPHERE_GPU_VM_NAME_SNAPSHOT
    export GOVC_INSECURE=1


    # install govc
    echo -n "Preparing: 'govc' existence check - "
    if util::cmd_exist govc; then
      echo "passed"
    else
      echo "installing govc"
      install_govc
    fi

    install_govc
    govc_poweroff_vm "$vm_name"
    govc_restore_vm_snapshot "$vm_name" "$vm_snapshot_name"
    govc_poweron_vm "$vm_name"
}


setup_gpu_test_env
