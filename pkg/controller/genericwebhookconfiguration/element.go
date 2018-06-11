/*
Copyright 2018 The Kubernetes Authors.

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

package genericwebhookconfiguration

import (
	"fmt"
	"reflect"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type webhookType string

const (
	mutatingWebhookType   webhookType = "mutating"
	validatingWebhookType webhookType = "validating"
)

// webhookConfigElement is a wrapper interface for MutatingWebhookConfiguration and
// ValidatingWebhookConfiguration to handle the common operations in the cert manager.
type webhookConfigElement interface {
	getType() webhookType

	getMetaAccessor() (metav1.Object, error)
	getTypeAccessor() (metav1.Type, error)

	deepCopy() webhookConfigElement
	deepEqual(webhookConfigElement) bool

	getWebhooks() []admissionregistrationv1beta1.Webhook

	getMutatingWebhookConfig() (*admissionregistrationv1beta1.MutatingWebhookConfiguration, error)
	getValidatingWebhookConfig() (*admissionregistrationv1beta1.ValidatingWebhookConfiguration, error)
}

func newWebhookElement(webhookConfiguration runtime.Object) (webhookConfigElement, error) {
	switch typed := webhookConfiguration.(type) {
	case *admissionregistrationv1beta1.MutatingWebhookConfiguration:
		return newMutatingWebhookElement(typed), nil
	case *admissionregistrationv1beta1.ValidatingWebhookConfiguration:
		return newValidatingWebhookElement(typed), nil
	default:
		return nil, fmt.Errorf("unsupported type: %T, only support v1beta1.MutatingWebhookConfiguration and v1beta1.ValidatingWebhookConfiguration", typed)
	}
}

func newMutatingWebhookElement(webhook *admissionregistrationv1beta1.MutatingWebhookConfiguration) *mutatingWebhookElement {
	return &mutatingWebhookElement{webhook: webhook}
}

type mutatingWebhookElement struct {
	webhook *admissionregistrationv1beta1.MutatingWebhookConfiguration
}

var _ webhookConfigElement = &mutatingWebhookElement{}

func (*mutatingWebhookElement) getType() webhookType {
	return mutatingWebhookType
}

func (e *mutatingWebhookElement) getMetaAccessor() (metav1.Object, error) {
	return meta.Accessor(e.webhook)
}

func (e *mutatingWebhookElement) getTypeAccessor() (metav1.Type, error) {
	return meta.TypeAccessor(e.webhook)
}

func (e *mutatingWebhookElement) deepCopy() webhookConfigElement {
	return &mutatingWebhookElement{webhook: e.webhook.DeepCopy()}
}

func (e *mutatingWebhookElement) deepEqual(other webhookConfigElement) bool {
	otherWebhookConfig, err := other.getMutatingWebhookConfig()
	if err != nil {
		return false
	}
	return reflect.DeepEqual(e.webhook, otherWebhookConfig)
}

func (e *mutatingWebhookElement) getWebhooks() []admissionregistrationv1beta1.Webhook {
	return e.webhook.Webhooks
}

func (e *mutatingWebhookElement) getMutatingWebhookConfig() (
	*admissionregistrationv1beta1.MutatingWebhookConfiguration, error) {
	return e.webhook, nil
}

func (e *mutatingWebhookElement) getValidatingWebhookConfig() (
	*admissionregistrationv1beta1.ValidatingWebhookConfiguration, error) {
	return nil, fmt.Errorf("unexpected getValidatingWebhookConfig")
}

func newValidatingWebhookElement(webhook *admissionregistrationv1beta1.ValidatingWebhookConfiguration) *validatingWebhookElement {
	return &validatingWebhookElement{webhook: webhook}
}

type validatingWebhookElement struct {
	webhook *admissionregistrationv1beta1.ValidatingWebhookConfiguration
}

var _ webhookConfigElement = &validatingWebhookElement{}

func (*validatingWebhookElement) getType() webhookType {
	return validatingWebhookType
}

func (e *validatingWebhookElement) getMetaAccessor() (metav1.Object, error) {
	return meta.Accessor(e.webhook)
}

func (e *validatingWebhookElement) getTypeAccessor() (metav1.Type, error) {
	return meta.TypeAccessor(e.webhook)
}

func (e *validatingWebhookElement) deepCopy() webhookConfigElement {
	return &validatingWebhookElement{webhook: e.webhook.DeepCopy()}
}

func (e *validatingWebhookElement) deepEqual(other webhookConfigElement) bool {
	otherWebhookConfig, err := other.getValidatingWebhookConfig()
	if err != nil {
		return false
	}
	return reflect.DeepEqual(e.webhook, otherWebhookConfig)
}

func (e *validatingWebhookElement) getWebhooks() []admissionregistrationv1beta1.Webhook {
	return e.webhook.Webhooks
}

func (e *validatingWebhookElement) getMutatingWebhookConfig() (
	*admissionregistrationv1beta1.MutatingWebhookConfiguration, error) {
	return nil, fmt.Errorf("unexpected getMutatingWebhookConfig")
}

func (e *validatingWebhookElement) getValidatingWebhookConfig() (
	*admissionregistrationv1beta1.ValidatingWebhookConfiguration, error) {
	return e.webhook, nil
}
