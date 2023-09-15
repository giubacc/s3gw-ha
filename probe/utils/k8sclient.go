// Copyright © 2023 SUSE LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/apis/core/helper"
)

type K8sClient struct {
	ClusterConfig *rest.Config
	ClientSet     *kubernetes.Clientset
}

func (k8s *K8sClient) Init() {
	// creates the in-cluster config
	var err error
	k8s.ClusterConfig, err = rest.InClusterConfig()
	if err != nil {
		Logger.Errorf("InClusterConfig: %s", err.Error())
	}
	k8s.ClientSet, err = kubernetes.NewForConfig(k8s.ClusterConfig)
	if err != nil {
		Logger.Errorf("NewForConfig: %s", err.Error())
	}
}

func (k8s *K8sClient) SetReplicasForDeployment(ns string, dName string, replicas int32) {
	dScaleOld, err := k8s.ClientSet.AppsV1().
		Deployments(ns).
		GetScale(context.TODO(), dName, metav1.GetOptions{})
	if err != nil {
		Logger.Errorf("GetScale: %s", err.Error())
	}

	dScale := *dScaleOld
	dScale.Spec.Replicas = replicas

	k8s.ClientSet.AppsV1().Deployments(ns).UpdateScale(context.TODO(), dName, &dScale, metav1.UpdateOptions{})
}

func (k8s *K8sClient) GetNodeNameList() (*[]string, error) {
	if list, err := k8s.ClientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{}); err == nil {
		var nodeNameList []string
		for _, node := range list.Items {
			nodeNameList = append(nodeNameList, node.Name)
		}
		return &nodeNameList, nil
	} else {
		return nil, err
	}
}

func (k8s *K8sClient) SetTaint(nodeName string, Key string, Value string, Effect v1.TaintEffect) error {
	Logger.Infof("Node: %s, applying taint: %s:%s %v", nodeName, Key, Value, Effect)
	taint := v1.Taint{Key: Key, Value: Value, Effect: Effect}
	return k8s.ApplyTaint(nodeName, &taint)
}

func (k8s *K8sClient) UnsetTaint(nodeName string, Key string, Value string, Effect v1.TaintEffect) error {
	Logger.Infof("Node: %s, removing taint: %s:%s %v", nodeName, Key, Value, Effect)
	taint := v1.Taint{Key: Key, Value: Value, Effect: Effect}
	return k8s.RemoveTaint(nodeName, &taint)
}

func (k8s *K8sClient) ApplyTaint(nodeName string, taint *v1.Taint) error {
	var (
		node    *v1.Node
		err     error
		updated bool
	)

	node, err = k8s.ClientSet.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	node, updated = k8s.addOrUpdateTaint(node, taint)
	if updated {
		if _, err = k8s.ClientSet.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{}); err != nil {
			Logger.Errorf("Failed to update node object: %v", err)
			return err
		}
	}
	return nil
}

func (k8s *K8sClient) RemoveTaint(nodeName string, taint *v1.Taint) error {
	node, err := k8s.ClientSet.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		Logger.Errorf("Failed to remove taint: %v", err)
		return err
	}
	var updated bool
	node, updated = k8s.removeTaint(node, taint)
	if updated {
		if _, err = k8s.ClientSet.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{}); err != nil {
			Logger.Errorf("Failed to update node object: %v", err)
			return err
		}
	}
	return nil
}

func (k8s *K8sClient) addOrUpdateTaint(node *v1.Node, taint *v1.Taint) (*v1.Node, bool) {
	newNode := node.DeepCopy()
	nodeTaints := newNode.Spec.Taints

	var newTaints []v1.Taint
	updated := false
	for i := range nodeTaints {
		if taint.MatchTaint(&nodeTaints[i]) {
			if helper.Semantic.DeepEqual(*taint, nodeTaints[i]) {
				return newNode, false
			}
			newTaints = append(newTaints, *taint)
			updated = true
			continue
		}

		newTaints = append(newTaints, nodeTaints[i])
	}

	if !updated {
		newTaints = append(newTaints, *taint)
	}

	newNode.Spec.Taints = newTaints
	return newNode, true
}

func (k8s *K8sClient) removeTaint(node *v1.Node, taint *v1.Taint) (*v1.Node, bool) {
	newNode := node.DeepCopy()
	nodeTaints := newNode.Spec.Taints
	if len(nodeTaints) == 0 {
		return newNode, false
	}

	if !k8s.taintExists(nodeTaints, taint) {
		return newNode, false
	}

	newTaints, _ := k8s.deleteTaint(nodeTaints, taint)
	newNode.Spec.Taints = newTaints
	return newNode, true
}

func (k8s *K8sClient) taintExists(taints []v1.Taint, taintToFind *v1.Taint) bool {
	for _, taint := range taints {
		if taint.MatchTaint(taintToFind) {
			return true
		}
	}
	return false
}

func (k8s *K8sClient) deleteTaint(taints []v1.Taint, taintToDelete *v1.Taint) ([]v1.Taint, bool) {
	newTaints := []v1.Taint{}
	deleted := false
	for i := range taints {
		if taintToDelete.MatchTaint(&taints[i]) {
			deleted = true
			continue
		}
		newTaints = append(newTaints, taints[i])
	}
	return newTaints, deleted
}
