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
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// webhookClient provide method(s) to deal with webhook configuration objects.
type webhookClient interface {
	update(webhookConfigElement) error
}

type delegatingWebhookClient struct {
	mutatingWebhookClient   *mutatingWebhookClient
	validatingWebhookClient *validatingWebhookClient
}

var _ webhookClient = &delegatingWebhookClient{}

func (m *delegatingWebhookClient) update(e webhookConfigElement) error {
	switch e.getType() {
	case mutatingWebhookType:
		return m.mutatingWebhookClient.update(e)
	case validatingWebhookType:
		return m.validatingWebhookClient.update(e)
	}
	return nil
}

func newMutatingWebhookClient(clientset kubernetes.Interface, informers informers.SharedInformerFactory) webhookClient {
	return &delegatingWebhookClient{
		mutatingWebhookClient: &mutatingWebhookClient{clientSet: clientset, informers: informers},
	}
}

func newValidatingWebhookClient(clientset kubernetes.Interface, informers informers.SharedInformerFactory) webhookClient {
	return &delegatingWebhookClient{
		validatingWebhookClient: &validatingWebhookClient{clientSet: clientset, informers: informers},
	}
}

type webhookClientImpl struct {
	clientSet kubernetes.Interface
	informers informers.SharedInformerFactory
}

type mutatingWebhookClient webhookClientImpl

var _ webhookClient = &mutatingWebhookClient{}

func (c *mutatingWebhookClient) update(e webhookConfigElement) error {
	w, err := e.getMutatingWebhookConfig()
	if err != nil {
		return err
	}
	_, err = c.clientSet.AdmissionregistrationV1beta1().
		MutatingWebhookConfigurations().Update(w)
	return err
}

type validatingWebhookClient webhookClientImpl

var _ webhookClient = &validatingWebhookClient{}

func (c *validatingWebhookClient) update(e webhookConfigElement) error {
	w, err := e.getValidatingWebhookConfig()
	if err != nil {
		return err
	}
	_, err = c.clientSet.AdmissionregistrationV1beta1().
		ValidatingWebhookConfigurations().Update(w)
	return err
}
