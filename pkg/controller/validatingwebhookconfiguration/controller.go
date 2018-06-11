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

package validatingwebhookconfiguration

import (
	"log"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"k8s.io/client-go/tools/record"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	admissionregistrationv1beta1informer "k8s.io/client-go/informers/admissionregistration/v1beta1"
	admissionregistrationv1beta1client "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"
	admissionregistrationv1beta1lister "k8s.io/client-go/listers/admissionregistration/v1beta1"

	"github.com/mengqiy/WebhookCertManager/pkg/inject/args"
)

// EDIT THIS FILE
// This files was created by "kubebuilder create resource" for you to edit.
// Controller implementation logic for ValidatingWebhookConfiguration resources goes here.

func (bc *ValidatingWebhookConfigurationController) Reconcile(k types.ReconcileKey) error {
	// INSERT YOUR CODE HERE
	log.Printf("Implement the Reconcile function on validatingwebhookconfiguration.ValidatingWebhookConfigurationController to reconcile %s\n", k.Name)
	return nil
}

// +kubebuilder:informers:group=admissionregistration,version=v1beta1,kind=ValidatingWebhookConfiguration
// +kubebuilder:rbac:groups=admissionregistration,resources=validatingwebhookconfigurations,verbs=get;watch;list
// +kubebuilder:controller:group=admissionregistration,version=v1beta1,kind=ValidatingWebhookConfiguration,resource=validatingwebhookconfigurations
type ValidatingWebhookConfigurationController struct {
	// INSERT ADDITIONAL FIELDS HERE
	validatingwebhookconfigurationLister admissionregistrationv1beta1lister.ValidatingWebhookConfigurationLister
	validatingwebhookconfigurationclient admissionregistrationv1beta1client.AdmissionregistrationV1beta1Interface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	validatingwebhookconfigurationrecorder record.EventRecorder
}

// ProvideController provides a controller that will be run at startup.  Kubebuilder will use codegeneration
// to automatically register this controller in the inject package
func ProvideController(arguments args.InjectArgs) (*controller.GenericController, error) {
	// INSERT INITIALIZATIONS FOR ADDITIONAL FIELDS HERE
	bc := &ValidatingWebhookConfigurationController{
		validatingwebhookconfigurationLister:   arguments.ControllerManager.GetInformerProvider(&admissionregistrationv1beta1.ValidatingWebhookConfiguration{}).(admissionregistrationv1beta1informer.ValidatingWebhookConfigurationInformer).Lister(),
		validatingwebhookconfigurationclient:   arguments.KubernetesClientSet.AdmissionregistrationV1beta1(),
		validatingwebhookconfigurationrecorder: arguments.CreateRecorder("ValidatingWebhookConfigurationController"),
	}

	// Create a new controller that will call ValidatingWebhookConfigurationController.Reconcile on changes to ValidatingWebhookConfigurations
	gc := &controller.GenericController{
		Name:             "ValidatingWebhookConfigurationController",
		Reconcile:        bc.Reconcile,
		InformerRegistry: arguments.ControllerManager,
	}
	if err := gc.Watch(&admissionregistrationv1beta1.ValidatingWebhookConfiguration{}); err != nil {
		return gc, err
	}

	// IMPORTANT:
	// To watch additional resource types - such as those created by your controller - add gc.Watch* function calls here
	// Watch function calls will transform each object event into a ValidatingWebhookConfiguration Key to be reconciled by the controller.
	//
	// **********
	// For any new Watched types, you MUST add the appropriate // +kubebuilder:informer and // +kubebuilder:rbac
	// annotations to the ValidatingWebhookConfigurationController and run "kubebuilder generate.
	// This will generate the code to start the informers and create the RBAC rules needed for running in a cluster.
	// See:
	// https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/controller#example-package
	// **********

	return gc, nil
}
