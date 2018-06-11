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

/*
Package genericwebhookconfiguration provides methods to ensure the webhooks can have
proper CA and certificate to work correctly.

You can create a GenericWebhookConfigurationController.
And then call Sync with a webhook configuration object.

	// A webhook configuration object to process.
	// One way to get a webhook configuration is to get from a k8s apiServer.
	var mmwhc := admissionregistrationv1beta1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			// TypeMeta fields
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"secret.certprovisioner.kubernetes.io/webhook-1": "namespace-bar/secret-foo",
				"secret.certprovisioner.kubernetes.io/webhook-2": "default/secret-baz",
			},
			// Other ObjectMeta fields
		},
		Webhooks: []admissionregistrationv1beta1.Webhook{
			{
				Name: "webhook-1",
				// Other fields
			},
			{
				Name: "webhook-2",
				// Other fields
			},
		},
	}

	// Build a certsHandlerFactory
	certsHandlerFactory := &SecretCertsReadWriterFactory{
		// Set KubernetesClientSet and KubernetesInformers

		// Set the method to get a CertProvisioner.
		GetCertProvisioner: func(commonName string) (certprovisioner.CertProvisioner, error) {
			return &certprovisioner.SelfSignedCertProvisioner{CommonName: commonName}, nil
		},
	}

	// Build a GenericWebhookConfigurationController
	genericWebhookConfigurationController := GenericWebhookConfigurationController{
		// Set KubernetesClientSet and KubernetesInformers
		CertsHandlerFactory: certsHandlerFactory,
	}
	// Sync for the certificate. It will provision the certificate and create an secret
	// named "secret-foo" in namespace "namespace-bar" for webhook "webhook-1".
	// Similarly, it will create an secret named "secret-baz" in namespace "default" for webhook "webhook-2".
	err := genericWebhookConfigurationController.Sync(mwc)
	if err != nil {
		// handler error
	}
*/
package genericwebhookconfiguration
