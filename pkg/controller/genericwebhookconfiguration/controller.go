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
	"bytes"
	"fmt"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/kubernetes-sigs/kubebuilder/pkg/webhook/certprovisioner"
)

type GenericWebhookConfigurationController struct {
	// kubernetesClientSet is a clientset to talk to Kuberntes apis
	KubernetesClientSet kubernetes.Interface
	// kubernetesInformers contains a Kubernetes informers factory
	KubernetesInformers informers.SharedInformerFactory

	// CertsHandlerFactory knows how to instantiate a CertsHandler.
	CertsHandlerFactory CertsHandlerFactory
	// GetCertProvisioner know how to get a CertProvisioner given the common name.
	GetCertProvisioner func(commonName string) (certprovisioner.CertProvisioner, error)
}

// Sync takes a runtime.Object which is expected to be either a MutatingWebhookConfiguration or
// a ValidatingWebhookConfiguration.
// It provisions the certs for each webhook in the webhookConfiguration, ensures the cert and CA are valid and
// update the CABundle in the webhook configuration if necessary.
func (bc *GenericWebhookConfigurationController) Sync(webhookConfiguration runtime.Object) error {
	certsHander, err := bc.CertsHandlerFactory.New(webhookConfiguration)
	if err != nil {
		return err
	}

	webhookClient, err := bc.newWebhookClient(webhookConfiguration)
	if err != nil {
		return err
	}

	webhookConfig, err := newWebhookElement(webhookConfiguration)
	if err != nil {
		return err
	}

	cloned := webhookConfig.deepCopy()
	webhooks := cloned.getWebhooks()
	for i := range webhooks {
		syncSecretWithWebhook(&webhooks[i], certsHander)
	}

	if webhookConfig.deepEqual(cloned) {
		return nil
	}
	return webhookClient.update(cloned)
}

// syncSecretWithWebhook ensures the certificate and CA exist and valid for the given webhook.
// syncSecretWithWebhook will modify the passed-in webhook.
func syncSecretWithWebhook(
	webhook *admissionregistrationv1beta1.Webhook, ch CertsHandler) error {
	webhookName := webhook.Name
	if ch.Skip(webhookName) {
		return nil
	}

	certs, err := ch.Read(webhookName)
	if apierrors.IsNotFound(err) {
		certs, err = ch.Write(webhookName)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Recreate the cert if it's invalid.
	if !validCertInSecret(certs) {
		certs, err = ch.Write(webhookName)
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
	}

	// Ensure the CA bundle in the webhook configuration has the signing CA.
	caBundle := webhook.ClientConfig.CABundle
	caCert := certs.CACert
	if !bytes.Contains(caBundle, caCert) {
		webhook.ClientConfig.CABundle = append(caBundle, caCert...)
	}
	return nil
}

func (bc *GenericWebhookConfigurationController) newWebhookClient(webhookConfiguration runtime.Object) (webhookClient, error) {
	switch typed := webhookConfiguration.(type) {
	case *admissionregistrationv1beta1.MutatingWebhookConfiguration:
		return newMutatingWebhookClient(bc.KubernetesClientSet, bc.KubernetesInformers), nil
	case *admissionregistrationv1beta1.ValidatingWebhookConfiguration:
		return newValidatingWebhookClient(bc.KubernetesClientSet, bc.KubernetesInformers), nil
	default:
		return nil, fmt.Errorf("unsupported type: %T, only support v1beta1.MutatingWebhookConfiguration and v1beta1.ValidatingWebhookConfiguration", typed)
	}
}
