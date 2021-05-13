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

// TopologyReleaseSpec defines the desired state of TopologyRelease
type TopologyReleaseSpec struct {
	Environment    string       `json:"environment,omitempty"`
	Version        string       `json:"version,omitempty"`
	BasedOnVersion string       `json:"basedOnVersion,omitempty"`
	ChangelogURL   string       `json:"changelogURL,omitempty"`
	Topology       []AppVersion `json:"topology,omitempty"`
}

type AppVersion struct {
	Name     string `json:"name,omitempty"`
	Version  string `json:"version,omitempty"`
	GitURL   string `json:"gitURL,omitempty"`
	Revision string `json:"revision,omitempty"`
}

// +kubebuilder:object:root=true
// +genclient
// +k8s:openapi-gen=true
// TopologyRelease is the Schema for the topologyReleases API
// +kubebuilder:printcolumn:name="Environment",type=string,JSONPath=`.spec.environment`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.metadata.namespace`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="BasedOnVersion",type=string,JSONPath=`.spec.basedOnVersion`
// +kubebuilder:printcolumn:name="ReleaseNotes",type=string,JSONPath=`.spec.changelogURL`
type TopologyRelease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TopologyReleaseSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// TopologyReleaseList contains a list of TopologyRelease
type TopologyReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TopologyRelease `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TopologyRelease{}, &TopologyReleaseList{})
}
