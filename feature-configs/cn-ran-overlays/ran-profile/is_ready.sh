#!/bin/bash

WORKER_NODES=$(oc get nodes -l "node-role.kubernetes.io/worker-cnf" -o name)

# Ensure numVfs is 4. If not, something is amiss
for node in ${WORKER_NODES//node/sriovnetworknodestate}
do
  [[ $(oc -n openshift-sriov-network-operator get ${node} -o json \
  | jq -r '.spec.interfaces[].numVfs | select(. != null)') -eq 4 ]] && \
  [[ $(oc -n openshift-sriov-network-operator get ${node} -o json \
  | jq -r '.status.interfaces[].numVfs | select(. != null)') -eq 4 ]]
done

oc wait ${CNF_MCP:-"mcp/worker-cnf"} --for condition=updated --timeout 1s
