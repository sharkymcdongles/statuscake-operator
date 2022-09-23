/*
Copyright 2022.

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

// UptimeTestSpec defines the desired state of UptimeTest
type UptimeTestSpec struct {
	// Type of check: DNS, HEAD, HTTP, PING, SMTP, SSH, or TCP
	// +kubebuilder:validation:Required
	// +kubebuilder:default:string=HTTP
	// +kubebuilder:validation:Enum:=DNS;HEAD;HTTP;PING;SMTP;SSH;TCP
	TestType string `json:"testtype"`
	// URL or IP address to check
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^https?:\/\/.+$`
	WebsiteUrl string `json:"websiteurl"` // TODO: Perhaps don't require http/s and auto set from ForceHttps?
	// Number of seconds between checks: 0, 30, 60, 300, 900, 1800, 3600, or 86400
	// +kubebuilder:validation:Required
	// +kubebuilder:default:int=300
	// +kubebuilder:validation:Enum=0;30;60;300;900;1800;3600;86400
	CheckRate int `json:"checkrate"`
	// Number of confirmation servers to confirm downtime before an alert is triggered
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:int=2
	// +kubebuilder:validation:Enum=0;1;2;3
	Confirmation int `json:"confirmation,omitempty"`
	// List of contact group numerical IDs
	// +kubebuilder:validation:Optional
	ContactGroups []string `json:"contactgroups,omitempty"`
	// Key, value pairs will be mapped to JSON object on backend. Represents headers to be sent when making requests
	// +kubebuilder:validation:Optional
	CustomHeader map[string]string `json:"customheader,omitempty"`
	// Whether to send an alert if the SSL certificate is soon to expire
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:boolean=true
	// +kubebuilder:validation:boolean=true;false
	EnableSslAlert bool `json:"enablesslalert,omitempty"`
	// Whether to follow redirects when testing. Disabled by default
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:boolean=true
	// +kubebuilder:validation:boolean=true;false
	FollowRedirects bool `json:"followredirects,omitempty"`
	// Force HTTPS for uptime check
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:boolean=true
	// +kubebuilder:validation:boolean=true;false
	ForceHttps bool `json:"forcehttps,omitempty"`
	// Whether the check should be run
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:boolean=false
	// +kubebuilder:validation:boolean=true;false
	Paused bool `json:"paused,omitempty"`
	// List of tags
	// +kubebuilder:validation:Optional
	Tags []string `json:"tags,omitempty"`
	// The number of seconds to wait to receive the first byte
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:int=5
	// +kubebuilder:validation:Minimum=5
	// +kubebuilder:validation:Maximum=70
	Timeout int `json:"timeout,omitempty"`
	// The number of minutes to wait before sending an alert
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:int=5
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=60
	TriggerRate int `json:"triggerrate,omitempty"`
	// Custom user agent string set when testing
	// +kubebuilder:validation:Optional
	UserAgent string `json:"useragent,omitempty"`
}

// UptimeTestStatus defines the observed state of UptimeTest
type UptimeTestStatus struct {
	// conditions represent the observations of uptimetests's current state.
	// +optional
	// +listType=map
	// +listMapKey=type
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []metav1.Condition `json:"conditions"`
}

// UptimeTestStatus condition types.
const (
	Updated = "TestUpdated"
	Created = "TestCreated"
)

// UptimeTest is the Schema for the uptimetests API
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.spec.websiteurl`
// +kubebuilder:printcolumn:name="Test ID",type=string,JSONPath=`.metadata.annotations.uptimetest\.sre\.mls\.io/statuscake-test-id`
// +kubebuilder:printcolumn:name="Paused",type=string,JSONPath=`.spec.paused`
// +kubebuilder:object:root=true
type UptimeTest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UptimeTestSpec   `json:"spec,omitempty"`
	Status UptimeTestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// UptimeTestList contains a list of UptimeTest
type UptimeTestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UptimeTest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&UptimeTest{}, &UptimeTestList{})
}
