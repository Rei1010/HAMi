#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -x

# This script runs e2e test against on kpanda control plane.
# You should prepare your environment in advance and following environment may be you need to set or use default one.
# - CONTROL_PLANE_KUBECONFIG: absolute path of control plane KUBECONFIG file.
#
# Usage: hack/run-e2e.sh

# Parameters:
#[CLUSTER_PREFIX]: Create prefix for king cluster name
#[E2E_TYPE]: PR or SCHEDULED = Judge run full e2etest and deploy all kind clusters

KUBECONFIG_PATH=${KUBECONFIG_PATH:-"${HOME}/.kube"}

CLUSTER_PREFIX=${1:-"kpanda"}
E2E_TYPE=${2:-"PR"}

HOST_CLUSTER_NAME="${CLUSTER_PREFIX}-host"
MAIN_KUBECONFIG=${MAIN_KUBECONFIG:-"${KUBECONFIG_PATH}/${HOST_CLUSTER_NAME}.config"}


# Install ginkgo
GOPATH=$(go env GOPATH | awk -F ':' '{print $1}')
export PATH=$PATH:$GOPATH/bin
# Run e2e
# kpanda-apiserver's svc is kpanda-apiserver
if [ "${E2E_TYPE}" == "HSE2E" ];then
  MAIN_KUBECONFIG="${E2E_CONFIG}"
  CLUSTER_PREFIX="kpanda"
else
  kubectl --kubeconfig="${MAIN_KUBECONFIG}" patch svc kpanda-apiserver -n kpanda-system -p '{"spec": {"type": "NodePort"}}'
  kubectl --kubeconfig="${MAIN_KUBECONFIG}" patch svc kpanda-ingress -n kpanda-system -p '{"spec": {"type": "NodePort"}}'
    # export redis port to nodeport enables the host to connect to the redis database.
  kubectl --kubeconfig="${MAIN_KUBECONFIG}" patch svc kpanda-redis-server  -n kpanda-system --type='json' -p '[{"op":"replace","path":"/spec/type","value":"NodePort"},{"op":"add","path":"/spec/ports/0/nodePort","value":30379}]'
fi
REDIS_ADDR=`kubectl --kubeconfig="${MAIN_KUBECONFIG}" get node -o=jsonpath={.items[0].status.addresses[0].address}`:30379
KUBESYSTEM_ID=`kubectl --kubeconfig="${MAIN_KUBECONFIG}" get ns kube-system  -o=jsonpath='{.metadata.uid}'`
if [[ "${E2E_TYPE}" == "CLUSTERLCM" ]] || [[ "${E2E_TYPE}" == "GPU" ]] ; then
  # This is e2etest that must be performed for each PR
  echo "skip for $E2E_TYPE"
elif [ "${E2E_TYPE}" == "HSE2E" ];then
    ginkgo -v --junit-report kpanda.xml -race --fail-fast --skip "\[bug\]" ./test/e2e/    -- --clusterprefix="${CLUSTER_PREFIX}" --kubeSystemID="${KUBESYSTEM_ID}" --IntegratedEnv="${IntegratedEnv}" --ControlPlaneName="${ControlPlaneName}"
else
  # This is e2etest that must be performed for each PR
  ginkgo -v -race --fail-fast --skip "\[bug\]" ./test/e2e/    -- --clusterprefix="${CLUSTER_PREFIX}" --kubeSystemID="${KUBESYSTEM_ID}"
  ginkgo -v -race --fail-fast --skip "\[bug\]" ./test/e2e/auth    -- --clusterprefix="${CLUSTER_PREFIX}" --kubeSystemID="${KUBESYSTEM_ID}" --redisAddr="${REDIS_ADDR}"
fi
# Test Engineer e2e-test (full e2e)
# Skip API testcases with bugs ([bug]tags)，Prevent e2e from failing to pass
# Empty judgment（When the change is detected by PRE2ESteps, run the accurate e2e, otherwise judge whether it is FULLE2E or SCHEDULED）
if [[ "${E2E_TYPE}" == "OFFLINE" ]]; then
    ginkgo -v -r -race -timeout=3h --fail-fast --skip "\[bug\]" --skip "\[bug-offline\]" --skip '\[gpu\]' ./test/e2e/api/  -- --clusterprefix="${CLUSTER_PREFIX}" --kubeSystemID="${KUBESYSTEM_ID}"
elif [ "${E2E_TYPE}" == "HSE2E" ];then
  echo -e "\033[31m-----🐮🐮🐮🐮🐮🐮-----------:e2etest type: ${E2E_TYPE}----🐮🐮🐮🐮🐮🐮---------\033[0m"
elif [ "${E2E_TYPE}" == "FULLE2E" -o "${E2E_TYPE}" == "SCHEDULED" ]; then
    ginkgo -v -r -race -timeout=3h --fail-fast --skip "\[bug\]" --skip '\[gpu\]' ./test/e2e/api/  -- --clusterprefix="${CLUSTER_PREFIX}" --kubeSystemID="${KUBESYSTEM_ID}"
    if [ "${E2E_TYPE}" == "SCHEDULED" ]; then
        ginkgo -v -r -timeout=3h -race --fail-fast --skip "\[bug\]"  ./test/e2e/compatibility/  -- --clusterprefix="${CLUSTER_PREFIX}" --kubeSystemID="${KUBESYSTEM_ID}"
    fi
elif [ "${E2E_TYPE}" == "CLUSTERLCM" ]; then
    ./hack/rollback-cluster.sh
    ginkgo -v -r -timeout=3h -race --fail-fast --skip "\[bug\]" ./test/e2e/clusterlcm/  -- --clusterprefix="${CLUSTER_PREFIX}" --kubeSystemID="${KUBESYSTEM_ID}"
elif [ "${E2E_TYPE}" == "GPU" ]; then
    ginkgo -v -r -timeout=3h -race --fail-fast --focus "${TARGET_PLATFORM}" --skip "\[bug\]" ./test/e2e/api/gpu/ -- --clusterprefix="${CLUSTER_PREFIX}" --kubeSystemID="${KUBESYSTEM_ID}"
else
    if [ ${PRE2ESteps:-nil} != "nil" ]; then
        # Array de duplication
        PRE2ESteps=($(awk -v RS=' ' '!a[$1]++' <<< ${PRE2ESteps[@]}))
        echo "The e2e test of the following modules will be performed：【${PRE2ESteps[@]}】"
        for step in ${PRE2ESteps[*]}
        do
            ginkgo -v -r -race --fail-fast --skip "\[bug\]" ./test/e2e/api/${step}/  -- --clusterprefix="${CLUSTER_PREFIX}" --kubeSystemID="${KUBESYSTEM_ID}"
        done
    fi
fi
