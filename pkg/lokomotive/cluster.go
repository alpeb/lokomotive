// Copyright 2021 The Lokomotive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lokomotive

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	// Period after which we assume cluster will not become reachable and we return timeout error to the user.
	clusterPingRetryTimeout = 5 * time.Minute
	// Number of seconds to wait between retires when waiting for cluster to become available.
	clusterPingRetryInterval = 10 * time.Second
	// Period after which we assume that nodes will never become ready and we return timeout error to the user.
	nodeReadinessRetryTimeout = 10 * time.Minute
	// Number of seconds to wait between retires when waiting for nodes to become ready.
	nodeReadinessRetryInterval = 10 * time.Second
)

type Cluster struct {
	KubeClient    *kubernetes.Clientset
	ExpectedNodes int
}

// NewCluster constructs and returns a new Cluster object.
func NewCluster(client *kubernetes.Clientset, expectedNodes int) *Cluster {
	return &Cluster{KubeClient: client, ExpectedNodes: expectedNodes}
}

func (cl *Cluster) Health() ([]v1.ComponentStatus, error) {
	cs, err := cl.KubeClient.CoreV1().ComponentStatuses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// For now we only show the status of etcd.
	var etcdComponents []v1.ComponentStatus

	for _, item := range cs.Items {
		if strings.HasPrefix(item.Name, "etcd") {
			etcdComponents = append(etcdComponents, item)
		}
	}

	return etcdComponents, nil
}

// NodeStatus represents the status of all nodes of a cluster.
type NodeStatus struct {
	nodeConditions map[string][]v1.NodeCondition
	expectedNodes  int
}

// GetNodeStatus returns the status for all running nodes or an error.
func (cl *Cluster) GetNodeStatus() (*NodeStatus, error) {
	n, err := cl.KubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	nodeConditions := make(map[string][]v1.NodeCondition)

	for _, node := range n.Items {
		nodeConditions[node.Name] = node.Status.Conditions
	}
	return &NodeStatus{
		nodeConditions: nodeConditions,
		expectedNodes:  cl.ExpectedNodes,
	}, nil
}

// Ready checks if all nodes are ready and returns false otherwise.
func (ns *NodeStatus) Ready() bool {
	if len(ns.nodeConditions) < ns.expectedNodes {
		return false
	}

	for _, conditions := range ns.nodeConditions {
		for _, condition := range conditions {
			if condition.Type == "Ready" && condition.Status != v1.ConditionTrue {
				return false
			}
		}
	}

	return true
}

// PrettyPrint prints Node statuses in a pretty way.
func (ns *NodeStatus) PrettyPrint() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)

	// Print the header.
	fmt.Fprintln(w, "\nNode\tReady\tReason\tMessage\t")

	// An empty line between header and the body.
	fmt.Fprintln(w, "\t\t\t\t")

	for node, conditions := range ns.nodeConditions {
		for _, condition := range conditions {
			if condition.Type == "Ready" {
				line := fmt.Sprintf(
					"%s\t%s\t%s\t%s\t",
					node, condition.Status, condition.Reason, condition.Message,
				)
				fmt.Fprintln(w, line)
			}
		}
	}
	if len(ns.nodeConditions) < ns.expectedNodes {
		line := fmt.Sprintf("%d nodes are missing", ns.expectedNodes-len(ns.nodeConditions))
		fmt.Fprintln(w, line)
	}

	w.Flush()
}

// ping Cluster to know when its endpoint can be used.
func (cl *Cluster) ping() (bool, error) {
	_, err := cl.KubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false, nil
	}
	return true, nil
}

// Verify checks cluster health and returns an error if some issues are detected.
func (cl *Cluster) Verify() error {
	fmt.Println("\nNow checking health and readiness of the cluster nodes ...")

	// Wait for cluster to become available.
	err := wait.PollImmediate(clusterPingRetryInterval, clusterPingRetryTimeout, cl.ping)
	if err != nil {
		return fmt.Errorf("pinging cluster for readiness: %w", err)
	}

	var ns *NodeStatus

	var nsErr error

	err = wait.PollImmediate(nodeReadinessRetryInterval, nodeReadinessRetryTimeout, func() (bool, error) {
		// Store the original error because Retry would stop too early if we forward it
		// and anyway overrides the error in case of timeout.
		ns, nsErr = cl.GetNodeStatus()
		if nsErr != nil {
			// To continue retrying, we don't set the error here.
			return false, nil
		}

		return ns.Ready(), nil // Retry if not ready.
	})

	if nsErr != nil {
		return fmt.Errorf("waiting for nodes: %w", nsErr)
	}

	if err != nil {
		return fmt.Errorf("not all nodes became ready within the allowed time")
	}

	ns.PrettyPrint()

	fmt.Println("\nSuccess - cluster is healthy and nodes are ready!")

	return nil
}
