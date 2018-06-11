package inject

import (
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	"github.com/mengqiy/WebhookCertManager/pkg/controller/mutatingwebhookconfiguration"
	"github.com/mengqiy/WebhookCertManager/pkg/controller/validatingwebhookconfiguration"
	"github.com/mengqiy/WebhookCertManager/pkg/inject/args"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func init() {

	// Inject Informers
	Inject = append(Inject, func(arguments args.InjectArgs) error {
		Injector.ControllerManager = arguments.ControllerManager

		// Add Kubernetes informers
		if err := arguments.ControllerManager.AddInformerProvider(&admissionregistrationv1beta1.MutatingWebhookConfiguration{}, arguments.KubernetesInformers.Admissionregistration().V1beta1().MutatingWebhookConfigurations()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&admissionregistrationv1beta1.ValidatingWebhookConfiguration{}, arguments.KubernetesInformers.Admissionregistration().V1beta1().ValidatingWebhookConfigurations()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1.Secret{}, arguments.KubernetesInformers.Core().V1().Secrets()); err != nil {
			return err
		}

		if c, err := mutatingwebhookconfiguration.ProvideController(arguments); err != nil {
			return err
		} else {
			arguments.ControllerManager.AddController(c)
		}
		if c, err := validatingwebhookconfiguration.ProvideController(arguments); err != nil {
			return err
		} else {
			arguments.ControllerManager.AddController(c)
		}
		return nil
	})

	// Inject CRDs
	// Inject PolicyRules
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{
			"",
		},
		Resources: []string{
			"secrets",
		},
		Verbs: []string{
			"create", "get", "list", "update", "watch",
		},
	})
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{
			"admissionregistration",
		},
		Resources: []string{
			"mutatingwebhookconfigurations",
		},
		Verbs: []string{
			"get", "list", "patch", "watch",
		},
	})
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{
			"admissionregistration",
		},
		Resources: []string{
			"validatingwebhookconfigurations",
		},
		Verbs: []string{
			"get", "list", "watch",
		},
	})
	// Inject GroupVersions
	Injector.RunFns = append(Injector.RunFns, func(arguments run.RunArguments) error {
		Injector.ControllerManager.RunInformersAndControllers(arguments)
		return nil
	})
}
