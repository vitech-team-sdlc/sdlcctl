/*


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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LargeTestExecutionSpec defines the desired state of LargeTestExecution
type LargeTestExecutionSpec struct {
	Image       string       `json:"image,omitempty"`
	Result      string       `json:"result,omitempty"`
	Environment string       `json:"environment,omitempty"`
	Namespace   string       `json:"namespace,omitempty"`
	Report      string       `json:"report,omitempty"`
	Time        string       `json:"time,omitempty"`
	Topology    []AppVersion `json:"topology,omitempty"`
}

type State string

const (
	StateAdded   State = "added"
	StateSame    State = "same"
	StateRemoved State = "removed"
	StateUpdated State = "updated"
)

type AppVersion struct {
	Name     string `json:"name,omitempty"`
	Version  string `json:"version,omitempty"`
	State    State  `json:"state,omitempty"`
	Revision string `json:"revision"`
}

// +kubebuilder:object:root=true
// +genclient
// +k8s:openapi-gen=true
// LargeTestExecution is the Schema for the largetestexecutions API
// +kubebuilder:printcolumn:name="Env",type=string,JSONPath=`.spec.environment`
// +kubebuilder:printcolumn:name="Ns",type=string,JSONPath=`.spec.namespace`
// +kubebuilder:printcolumn:name="Report",type=string,JSONPath=`.spec.report`
// +kubebuilder:printcolumn:name="Result",type=string,JSONPath=`.spec.result`
// +kubebuilder:printcolumn:name="Time",type=string,JSONPath=`.spec.time`
type LargeTestExecution struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec LargeTestExecutionSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// LargeTestExecutionList contains a list of LargeTestExecution
type LargeTestExecutionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LargeTestExecution `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LargeTestExecution{}, &LargeTestExecutionList{})
}
