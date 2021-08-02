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

package util

import (
	"encoding/json"
	"fmt"
)

// NodeAffinity is a struct that other components can use to define the HCL format of NodeAffinity
// in Kubernetes PodSpec.
type NodeAffinity struct {
	Key      string   `hcl:"key" json:"key,omitempty"`
	Operator string   `hcl:"operator" json:"operator,omitempty"`
	Values   []string `hcl:"values,optional" json:"values,omitempty"`
}

// RenderNodeAffinity takes a list of NodeAffinity.
// It returns a json string and an error if any.
func RenderNodeAffinity(n []NodeAffinity) (string, error) {
	if len(n) == 0 {
		return "", nil
	}

	b, err := json.Marshal(n)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

type Toleration struct {
	Key               string `hcl:"key,optional" json:"key,omitempty"`
	Effect            string `hcl:"effect,optional" json:"effect,omitempty"`
	Operator          string `hcl:"operator,optional" json:"operator,omitempty"`
	Value             string `hcl:"value,optional" json:"value,omitempty"`
	TolerationSeconds int64  `hcl:"toleration_seconds,optional" json:"tolerationSeconds,omitempty"`
}

// RenderTolerations takes a list of tolerations.
// It returns a json string and an error if any.
func RenderTolerations(t []Toleration) (string, error) {
	if len(t) == 0 {
		return "", nil
	}

	for _, toleration := range t {
		if err := validateToleration(toleration); err != nil {
			return "", fmt.Errorf("toleration validation failed: %v", err)
		}
	}

	b, err := json.Marshal(t)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func validateToleration(t Toleration) error {
	// If TolerationSeconds is set then; Effect must be `NoExecute`
	if t.TolerationSeconds != 0 && t.Effect != "NoExecute" {
		return fmt.Errorf("`effect` must be `NoExecute` as `toleration_seconds` is set: got %s", t.Effect)
	}

	return nil
}

// NodeSelector is a type used when defining node selector for the pod spec.
type NodeSelector map[string]string

// Render renders NodeSelector into a json string.
func (n *NodeSelector) Render() (string, error) {
	if len(*n) == 0 {
		return "", nil
	}

	b, err := json.Marshal(n)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// ResourceRequirements allows user to specify resource requests and/or limits.
type ResourceRequirements struct {
	Requests *ResourceList `hcl:"requests,block" json:"requests,omitempty"`
	Limits   *ResourceList `hcl:"limits,block" json:"limits,omitempty"`
}

// ResourceList allows user to specify CPU and memory.
type ResourceList struct {
	CPU    string `hcl:"cpu,optional" json:"cpu,omitempty"`
	Memory string `hcl:"memory,optional" json:"memory,omitempty"`
}

// RenderResourceRequirements takes a list of ResourceRequirements.
// It returns a json string and an error if any.
func RenderResourceRequirements(r *ResourceRequirements) (string, error) {
	if r == nil {
		return "", nil
	}

	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
