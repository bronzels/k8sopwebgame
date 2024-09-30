/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WebGameSpec defines the desired state of WebGame
type WebGameSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	DisplayName string `json:"displayName"`
	GameType    string `json:"gameType"`
	// +kubebuilder:default:=localhost
	Domain string `json:"domain"`
	// +kubebuilder:default:=/
	IndexPage    string             `json:"indexPage"`
	IngressClass string             `json:"ingressClass"`
	ServerPort   intstr.IntOrString `json:"serverPort"`
	Replicas     *int32             `json:"replicas"`
	Image        string             `json:"image"`
	// +kubebuilder:validation:Optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Foo is an example field of WebGame. Edit webgame_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// WebGameStatus defines the observed state of WebGame
type WebGameStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	DeploymentStatus appsv1.DeploymentStatus `json:"deploymentStatus"`
	GameAddress      string                  `json:"gameAddress,omitempty"`
	ClusterIP        string                  `json:"clusterIP,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// WebGame is the Schema for the webgames API
type WebGame struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebGameSpec   `json:"spec,omitempty"`
	Status WebGameStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WebGameList contains a list of WebGame
type WebGameList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WebGame `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WebGame{}, &WebGameList{})
}
