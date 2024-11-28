#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -x

# This script plays as a reference script to install kpanda helm release
#
# Usage: bash hack/deploy.sh v0.0.1-8-gee28ca5 latest  remote_kube_cluster.conf
#        Parameter 1: helm release version

# specific a helm package version
HELM_VER=${1:-"v2.4.1"}
HELM_NAME=${2:-"hami-charts"}
HELM_REPO=${3:-"https://project-hami.github.io/HAMi/"}
TARGET_NS=${4:-"hami-system"}
HAMI_ALIAS="hami"


REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${REPO_ROOT}"

source "${REPO_ROOT}"/hack/util.sh

# install helm
echo -n "Preparing: 'helm' existence check - "
if util::cmd_exist helm; then
  echo "passed"
else
  echo "installing helm"
  util::install_helm
fi

# add repo locally
helm repo add "${HELM_NAME}" "${HELM_REPO}" --force-update --kubeconfig "${KUBE_CONF}"
helm repo update --kubeconfig "${KUBE_CONF}"

kubectl label nodes gpu-master gpu=on --kubeconfig "${KUBE_CONF}"

# install or upgrade
util::exec_cmd helm --debug upgrade --install --create-namespace --cleanup-on-fail \
             "${HAMI_ALIAS}"     "${HELM_NAME}"/"${HAMI_ALIAS}"  \
             -n "${TARGET_NS}"  --version ${HELM_VER} \
             --kubeconfig "${KUBE_CONF}"

## check it
#util::check_helm_deployment "${TARGET_NS}" ""${KUBE_CONF}""
