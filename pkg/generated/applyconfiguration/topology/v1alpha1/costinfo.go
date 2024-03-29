/*
Copyright The Kubernetes Authors.

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
// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

// CostInfoApplyConfiguration represents an declarative configuration of the CostInfo type for use
// with apply.
type CostInfoApplyConfiguration struct {
	Name  *string `json:"name,omitempty"`
	Value *int64  `json:"value,omitempty"`
}

// CostInfoApplyConfiguration constructs an declarative configuration of the CostInfo type for use with
// apply.
func CostInfo() *CostInfoApplyConfiguration {
	return &CostInfoApplyConfiguration{}
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *CostInfoApplyConfiguration) WithName(value string) *CostInfoApplyConfiguration {
	b.Name = &value
	return b
}

// WithValue sets the Value field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Value field is set to the value of the last call.
func (b *CostInfoApplyConfiguration) WithValue(value int64) *CostInfoApplyConfiguration {
	b.Value = &value
	return b
}
