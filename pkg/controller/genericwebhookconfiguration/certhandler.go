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
	"net/url"
	"strings"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/kubernetes-sigs/kubebuilder/pkg/webhook/certprovisioner"
)

var (
	// Use an annotation in the following format:
	// secret.certprovisioner.kubernetes.io/<webhook-name>: <secret-namespace>/<secret-name>
	// the webhook cert manager library will provision the certificate for the webhook by
	// storing it in the specified secret.
	SecretCertInjectionAnnotationKeyPrefix = "secret.certprovisioner.kubernetes.io/"

	// Use an annotation in the following format:
	// local.certprovisioner.kubernetes.io/<webhook-name>: path/to/certs/
	// the webhook cert manager library will provision the certificate for the webhook by
	// storing it under the specified path.
	// format: local.certprovisioner.kubernetes.io/webhookName: path/to/certs/
	LocalCertInjectionAnnotationKeyPrefix = "local.certprovisioner.kubernetes.io/"

	CACertName     = "ca-cert.pem"
	ServerKeyName  = "key.pem"
	ServerCertName = "cert.pem"
)

// CertsHandlerFactory builds CertsHandler.
type CertsHandlerFactory interface {
	// New instantiates a CertHandler with a webhookConfiguration object.
	New(webhookConfiguration runtime.Object) (CertsHandler, error)
}

// CertsHandler provides methods for handling certificates for webhook.
type CertsHandler interface {
	// Skip returns if the webhook should be skipped.
	Skip(webhookName string) bool
	// Read reads a wehbook name and returns the certs for it.
	Read(webhookName string) (*certprovisioner.Certs, error)
	// Write writes the certs and return the certs it wrote.
	Write(webhookName string) (*certprovisioner.Certs, error)
}

type SecretCertsReadWriterFactory struct {
	KubernetesClientSet kubernetes.Interface
	KubernetesInformers informers.SharedInformerFactory
	// Method to get a CertProvisioner given the common name.
	GetCertProvisioner func(commonName string) (certprovisioner.CertProvisioner, error)
}

var _ CertsHandlerFactory = &SecretCertsReadWriterFactory{}

// New takes a runtime.Object which is expected to be either a MutatingWebhookConfiguration or
// a ValidatingWebhookConfiguration, it returns a CertHandler for this webhook configuration.
// It will return an error if the passed-in type is not a webhook configuration type.
func (s *SecretCertsReadWriterFactory) New(webhookConfiguration runtime.Object) (CertsHandler, error) {
	var element webhookConfigElement
	switch typed := webhookConfiguration.(type) {
	case *admissionregistrationv1beta1.MutatingWebhookConfiguration:
		element = newMutatingWebhookElement(typed)
	case *admissionregistrationv1beta1.ValidatingWebhookConfiguration:
		element = newValidatingWebhookElement(typed)
	default:
		return nil, fmt.Errorf("unsupported type: %T, only support v1beta1.MutatingWebhookConfiguration and v1beta1.ValidatingWebhookConfiguration", typed)
	}
	webhookToSecret := map[string]apitypes.NamespacedName{}
	accessor, err := meta.Accessor(webhookConfiguration)
	if err != nil {
		return nil, err
	}
	annotations := accessor.GetAnnotations()
	if annotations == nil {
		return nil, nil
	}
	for k, v := range annotations {
		if strings.HasPrefix(k, SecretCertInjectionAnnotationKeyPrefix) {
			webhookName := strings.TrimPrefix(k, SecretCertInjectionAnnotationKeyPrefix)
			webhookToSecret[webhookName] = apitypes.NewNamespacedNameFromString(v)
		}
	}
	ch := &secretCertsReadWriter{
		kubernetesClientSet: s.KubernetesClientSet,
		kubernetesInformers: s.KubernetesInformers,
		webhookConfig:       element,
		webhookToSecrets:    webhookToSecret,
		getCertProvisioner:  s.GetCertProvisioner,
	}

	return ch, nil
}

type secretCertsReadWriter struct {
	kubernetesClientSet kubernetes.Interface
	kubernetesInformers informers.SharedInformerFactory

	// The webhookConfiguration it is going to handle.
	webhookConfig webhookConfigElement
	// A map from wehbook name to the individual webhook.
	webhookMap map[string]*admissionregistrationv1beta1.Webhook
	// A map from webhook name to the service namespace and name.
	webhookToSecrets map[string]apitypes.NamespacedName

	getCertProvisioner func(commonName string) (certprovisioner.CertProvisioner, error)
}

var _ CertsHandler = &secretCertsReadWriter{}

func (s *secretCertsReadWriter) Skip(webhookName string) bool {
	_, found := s.webhookToSecrets[webhookName]
	return !found
}

func (s *secretCertsReadWriter) Write(webhookName string) (
	*certprovisioner.Certs, error) {
	sec, found := s.webhookToSecrets[webhookName]
	if !found {
		return nil, fmt.Errorf("failed to find the secret name by the webhook name: %q", webhookName)
	}

	webhook := s.webhookMap[webhookName]
	commonName, err := webhookClientConfigToCommonName(&webhook.ClientConfig)
	if err != nil {
		return nil, err
	}
	cp, err := s.getCertProvisioner(commonName)
	if err != nil {
		return nil, err
	}

	secret, err := s.kubernetesInformers.Core().V1().Secrets().Lister().Secrets(sec.Namespace).Get(sec.Name)
	if apierrors.IsNotFound(err) {
		certs, err := cp.ProvisionServingCert()
		if err != nil {
			return nil, err
		}
		secret = certsToSecret(certs, sec)
		// TODO fix and enable it
		//err = setOwnerRef(secret, webhookConfig)
		_, err = s.kubernetesClientSet.CoreV1().Secrets(sec.Namespace).Create(secret)
		return certs, err
	} else if err != nil {
		return nil, err
	}

	certs, err := secretToCerts(secret)
	if err != nil {
		return nil, err
	}
	// Recreate the cert if it's invalid.
	if !validCertInSecret(certs) {
		certs, err = cp.ProvisionServingCert()
		if err != nil {
			return nil, err
		}
		secret = certsToSecret(certs, sec)
		// TODO fix and enable it
		//err = setOwnerRef(secret, webhookConfig)
		_, err = s.kubernetesClientSet.CoreV1().Secrets(sec.Namespace).Update(secret)
		return certs, err
	}
	return certs, nil
}

func (s *secretCertsReadWriter) Read(webhookName string) (*certprovisioner.Certs, error) {
	sec, found := s.webhookToSecrets[webhookName]
	if !found {
		return nil, fmt.Errorf("failed to find the secret name by the webhook name: %q", webhookName)
	}
	secret, err := s.kubernetesInformers.Core().V1().Secrets().Lister().Secrets(sec.Namespace).Get(sec.Name)
	if err != nil {
		return nil, err
	}
	return secretToCerts(secret)
}

// Mark the webhook as the owner of the secret by setting the ownerReference in the secret.
func setOwnerRef(secret, webhookConfig webhookConfigElement) error {
	accessor, err := webhookConfig.getMetaAccessor()
	// TODO: typeAccessor.GetAPIVersion() and typeAccessor.GetKind() returns empty apiVersion and Kind, fix it.
	typeAccessor, err := webhookConfig.getTypeAccessor()
	if err != nil {
		return err
	}
	blockOwnerDeletion := false
	// Due to
	// https://github.com/kubernetes/kubernetes/blob/5da925ad4fd070e687dc5255c177d5e7d542edd7/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/controller_ref.go#L35
	isController := true
	ownerRef := metav1.OwnerReference{
		APIVersion:         typeAccessor.GetAPIVersion(),
		Kind:               typeAccessor.GetKind(),
		Name:               accessor.GetName(),
		UID:                accessor.GetUID(),
		BlockOwnerDeletion: &blockOwnerDeletion,
		Controller:         &isController,
	}
	secretAccessor, err := meta.Accessor(secret)
	if err != nil {
		return err
	}
	secretAccessor.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
	return nil
}

func validCertInSecret(certs *certprovisioner.Certs) bool {
	// TODO: 1) validate the key and the cert are valid pair e.g. call crypto/tls.X509KeyPair()
	// 2) validate the cert with the CA cert
	// 3) validate the cert is for a certain DNSName
	// e.g.
	// c, err := tls.X509KeyPair(cert, key)
	// err := c.Verify(options)

	return true
}

func secretToCerts(secret *corev1.Secret) (*certprovisioner.Certs, error) {
	checkList := []string{CACertName, ServerCertName, ServerKeyName}
	for _, key := range checkList {
		if _, ok := secret.Data[key]; !ok {
			return nil, fmt.Errorf("failed to find required key: %q in the secret", key)
		}
	}
	return &certprovisioner.Certs{
		CACert: secret.Data[CACertName],
		Cert:   secret.Data[ServerCertName],
		Key:    secret.Data[ServerKeyName],
	}, nil
}

func certsToSecret(certs *certprovisioner.Certs, sec apitypes.NamespacedName) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: sec.Namespace,
			Name:      sec.Name,
		},
		Data: map[string][]byte{
			CACertName:     certs.CACert,
			ServerKeyName:  certs.Key,
			ServerCertName: certs.Cert,
		},
	}
}

func webhookClientConfigToCommonName(config *admissionregistrationv1beta1.WebhookClientConfig) (string, error) {
	if config.Service != nil && config.URL != nil {
		return "", fmt.Errorf("service and URL can't be set at the same time in a webhook: %#v", config)
	}
	if config.Service == nil && config.URL == nil {
		return "", fmt.Errorf("one of service and URL need to be set in a webhook: %#v", config)
	}
	if config.Service != nil {
		return certprovisioner.ServiceToCommonName(config.Service.Namespace, config.Service.Name), nil
	}
	if config.URL != nil {
		u, err := url.Parse(*config.URL)
		return u.Host, err
	}
	return "", nil
}

//type FSCertsReadWriterFactory struct{}
//
//var _ CertsHandlerFactory = &FSCertsReadWriterFactory{}
//
//func (s *FSCertsReadWriterFactory) New(webhookConfiguration runtime.Object) (CertsHandler, error) {
//	// TODO: implement this
//	return nil, nil
//}
//
//// Write the local FS.
//// This is designed for running as static pod on the master node.
//type fsCertsReadWriter struct {
//	// The webhookConfiguration it is going to handle.
//	webhookConfig webhookConfigElement
//	// A map from webhook name to the path to write the certificates.
//	WebhookToPath map[string]string
//
//	getCertProvisioner func(commonName string) (certprovisioner.CertProvisioner, error)
//}
//
//var _ CertsHandler = &fsCertsReadWriter{}
//
//func (s *fsCertsReadWriter) Skip(webhookName string) bool {
//	// TODO: implement this
//	return true
//}
//
//func (s *fsCertsReadWriter) Write(webhookName string) (
//	*certprovisioner.Certs, error) {
//	// TODO: implement this
//	return nil, nil
//}
//
//func (s *fsCertsReadWriter) Read(webhookName string) (*certprovisioner.Certs, error) {
//	// TODO: implement this
//	return nil, nil
//}
