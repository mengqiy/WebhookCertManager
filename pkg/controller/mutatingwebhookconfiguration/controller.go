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

package mutatingwebhookconfiguration

import (
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/eventhandlers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"github.com/kubernetes-sigs/kubebuilder/pkg/webhook/certprovisioner"

	"github.com/mengqiy/WebhookCertManager/pkg/controller/genericwebhookconfiguration"
	"github.com/mengqiy/WebhookCertManager/pkg/inject/args"
)

// EDIT THIS FILE
// This files was created by "kubebuilder create resource" for you to edit.
// Controller implementation logic for MutatingWebhookConfiguration resources goes here.

func (bc *MutatingWebhookConfigurationController) Reconcile(k types.ReconcileKey) error {
	webhook, err := bc.KubernetesInformers.Admissionregistration().V1beta1().MutatingWebhookConfigurations().Lister().Get(k.Name)
	if err != nil {
		return err
	}
	return bc.GenericWebhookConfigurationController.Sync(webhook)
}

// +kubebuilder:informers:group=core,version=v1,kind=secret
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=create;update;get;watch;list
// +kubebuilder:informers:group=admissionregistration,version=v1beta1,kind=MutatingWebhookConfiguration
// +kubebuilder:rbac:groups=admissionregistration,resources=mutatingwebhookconfigurations,verbs=get;watch;list;update
// +kubebuilder:controller:group=admissionregistration,version=v1beta1,kind=MutatingWebhookConfiguration,resource=mutatingwebhookconfigurations
type MutatingWebhookConfigurationController struct {
	args.InjectArgs

	genericwebhookconfiguration.GenericWebhookConfigurationController

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	mutatingwebhookconfigurationrecorder record.EventRecorder
}

// ProvideController provides a controller that will be run at startup.  Kubebuilder will use codegeneration
// to automatically register this controller in the inject package
func ProvideController(arguments args.InjectArgs) (*controller.GenericController, error) {
	// INSERT INITIALIZATIONS FOR ADDITIONAL FIELDS HERE

	// TODO: add labels to some whitelisted namespace to run the webhook
	// so it won't block its own pod creation.

	bc := &MutatingWebhookConfigurationController{
		InjectArgs: arguments,
		GenericWebhookConfigurationController: genericwebhookconfiguration.GenericWebhookConfigurationController{
			KubernetesClientSet: arguments.KubernetesClientSet,
			KubernetesInformers: arguments.KubernetesInformers,
			CertsHandlerFactory: &genericwebhookconfiguration.SecretCertsReadWriterFactory{
				KubernetesClientSet: arguments.KubernetesClientSet,
				KubernetesInformers: arguments.KubernetesInformers,
				GetCertProvisioner: func(commonName string) (certprovisioner.CertProvisioner, error) {
					return &certprovisioner.SelfSignedCertProvisioner{CommonName: commonName}, nil
				},
			},
		},
		mutatingwebhookconfigurationrecorder: arguments.CreateRecorder("MutatingWebhookConfigurationController"),
	}

	// Create a new controller that will call MutatingWebhookConfigurationController.Reconcile on changes to MutatingWebhookConfigurations
	gc := &controller.GenericController{
		Name:             "MutatingWebhookConfigurationController",
		Reconcile:        bc.Reconcile,
		InformerRegistry: arguments.ControllerManager,
	}

	if err := gc.Watch(&admissionregistrationv1beta1.MutatingWebhookConfiguration{}); err != nil {
		return gc, err
	}

	// IMPORTANT:
	// To watch additional resource types - such as those created by your controller - add gc.Watch* function calls here
	// Watch function calls will transform each object event into a MutatingWebhookConfiguration Key to be reconciled by the controller.
	//
	// **********
	// For any new Watched types, you MUST add the appropriate // +kubebuilder:informer and // +kubebuilder:rbac
	// annotations to the MutatingWebhookConfigurationController and run "kubebuilder generate.
	// This will generate the code to start the informers and create the RBAC rules needed for running in a cluster.
	// See:
	// https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/controller#example-package
	// **********

	secretLookup := func(k types.ReconcileKey) (interface{}, error) {
		return bc.KubernetesInformers.Core().V1().Secrets().Lister().Secrets(k.Namespace).Get(k.Name)
	}
	if err := gc.WatchControllerOf(&corev1.Secret{},
		eventhandlers.Path{secretLookup}); err != nil {
		return gc, err
	}

	return gc, nil
}
