/*
Copyright Confidential Containers Contributors.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Deployment Type string determines the way to deploy the KBS
// +enum
type DeploymentType string

const (
	// DeploymentTypeAllInOne: all the KBS components will be deployed in the same container
	DeploymentTypeAllInOne DeploymentType = "AllInOneDeployment"

	// DeploymentTypeMicroservices: all the KBS components will be deployed in separate containers
	DeploymentTypeMicroservices DeploymentType = "MicroservicesDeployment"
)

// TdxConfigSpec defines the desired state for TDX configuration
type TdxConfigSpec struct {
	// kbsTdxConfigMapName is the name of the configmap containing sgx_default_qcnl.conf file
	// +optional
	KbsTdxConfigMapName string `json:"kbsTdxConfigMapName,omitempty"`
}

// KbsConfigSpec defines the desired state of KbsConfig
type KbsConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// KbsConfigMapName is the name of the configmap that contains the KBS configuration
	KbsConfigMapName string `json:"kbsConfigMapName,omitempty"`

	// KbsAsConfigMapName is the name of the configmap that contains the KBS AS configuration
	// Required only when MicroservicesDeployment is set
	// +optional
	KbsAsConfigMapName string `json:"kbsAsConfigMapName,omitempty"`

	// KbsRvpsConfigMapName is the name of the configmap that contains the KBS RVPS configuration
	// Required only when MicroservicesDeployment is set
	// +optional
	KbsRvpsConfigMapName string `json:"kbsRvpsConfigMapName,omitempty"`

	// kbsRvpsRefValuesConfigMapName is the name of the configmap that contains the RVPS reference values
	KbsRvpsRefValuesConfigMapName string `json:"kbsRvpsRefValuesConfigMapName,omitempty"`

	// KbsAuthSecretName is the name of the secret that contains the KBS auth secret
	KbsAuthSecretName string `json:"kbsAuthSecretName,omitempty"`

	// KbsServiceType is the type of service to create for KBS
	// Default value is ClusterIP
	// +optional
	KbsServiceType corev1.ServiceType `json:"kbsServiceType,omitempty"`

	// KbsDeploymentType is the type of KBS deployment
	// It can assume one of the following values:
	//    AllInOneDeployment: all the KBS components will be deployed in the same container
	//    MicroservicesDeployment: all the KBS components will be deployed in separate containers
	// +kubebuilder:validation:Enum=AllInOneDeployment;MicroservicesDeployment
	// Default value is AllInOneDeployment
	// +optional
	KbsDeploymentType DeploymentType `json:"kbsDeploymentType,omitempty"`

	// KbsHttpsKeySecretName is the name of the secret that contains the KBS https private key
	KbsHttpsKeySecretName string `json:"kbsHttpsKeySecretName,omitempty"`

	// KbsHttpsCertSecretName is the name of the secret that contains the KBS https certificate
	KbsHttpsCertSecretName string `json:"kbsHttpsCertSecretName,omitempty"`

	// KbsSecretResources is an array of secret names that contain the keys required by clients
	// +optional
	KbsSecretResources []string `json:"kbsSecretResources,omitempty"`

	// kbsResourcePolicyConfigMapName is the name of the configmap that contains the Resource Policy
	// +optional
	KbsResourcePolicyConfigMapName string `json:"kbsResourcePolicyConfigMapName,omitempty"`

	// tdxConfigSpec is the struct that hosts the TDX specific configuration
	// +optional
	TdxConfigSpec TdxConfigSpec `json:"tdxConfigSpec,omitempty"`
}

// KbsConfigStatus defines the observed state of KbsConfig
type KbsConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// IsReady is true when the KBS configuration is ready
	IsReady bool `json:"isReady,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KbsConfig is the Schema for the kbsconfigs API
type KbsConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KbsConfigSpec   `json:"spec,omitempty"`
	Status KbsConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KbsConfigList contains a list of KbsConfig
type KbsConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KbsConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KbsConfig{}, &KbsConfigList{})
}
