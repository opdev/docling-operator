/*
Copyright 2025.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DoclingServeSpec defines the desired state of DoclingServe.
type DoclingServeSpec struct {
	// +kubebuilder:validation:Required
	APIServer *APIServer `json:"apiServer"`

	// +kubebuilder:validation:Required,name="Engine"
	Engine *Engine `json:"engine"`

	// +kubebuilder:validation:Optional,name="Route"
	Route *Route `json:"route,omitempty"`
}

// APIServer configures a docling-serve workload
type APIServer struct {
	// Image specifics which docling-serve container image to deploy.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// EnableUI determines whether to run the docling-serve ui.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable UI",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// +kubebuilder:validation:Optional
	EnableUI bool `json:"enableUI,omitempty"`

	// Instances represents the desired number of docling-serve workloads to create.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Instance Count",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:podCount"}
	// +kubebuilder:default=1
	Instances int32 `json:"instances,omitempty"`

	// ConfigMapName represents the config map name that contains additional configurations.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="ConfigMap Name",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	ConfigMapName string `json:"configMapName,omitempty"`
}

// Route configures an OpenShift route, exposed Docling API outside the cluster.
type Route struct {
	// Enabled determines whether to create a route.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable Route",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// +kubebuilder:validation:Optional
	Enabled bool `json:"enabled,omitempty"`
}

// The below Engine struct has XValidation logic that is written to provide mutual exclusivity between `Local` and `KFP` structs.
// Currently, K8s' CEL implementation does not support `OneOf` logic. When the below issue is implemented, we can simplify the logic to be `OneOf`
// https://github.com/kubernetes-sigs/controller-tools/issues/461

// Engine defines which type of docling-serve compute engine to deploy. The selected engine will run all the async jobs.
// +kubebuilder:validation:XValidation:rule="(has(self.local) && !has(self.kfp)) || (!has(self.local) && has(self.kfp))", message="Only a Local or KFP Engine are allowed to be configured not both"
type Engine struct {
	Local *Local `json:"local,omitempty"`
	KFP   *KFP   `json:"kfp,omitempty"`
}

// Local configures the docling-serve engine.
type Local struct {
	// NumWorkers the desired number workers/threads processing the incoming tasks.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Number of Workers",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:podCount"}
	// +kubebuilder:validation:Required
	// +kubebuilder:default=2
	NumWorkers int32 `json:"numWorkers"`
}

// KFP configures a Kubeflow Pipeline engine.
type KFP struct {
	// The Kubeflow Pipeline endpoint location, example: https://NAME.NAMESPACE.svc.cluster.local:8888
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Kubeflow Pipeline Endpoint",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	Endpoint string `json:"endpoint"`
}

// DoclingServeStatus defines the observed state of DoclingServe
type DoclingServeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions describe the state of the operator's reconciliation functionality.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +optional
	// Conditions is a list of conditions related to operator reconciliation
	Conditions []metav1.Condition `json:"conditions,omitempty"  patchStrategy:"merge" patchMergeKey:"type"`

	// ObservedGeneration is the generation last observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DoclingServe is the Schema for the doclingserves API
type DoclingServe struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DoclingServeSpec   `json:"spec,omitempty"`
	Status DoclingServeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DoclingServeList contains a list of DoclingServe
type DoclingServeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DoclingServe `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DoclingServe{}, &DoclingServeList{})
}
