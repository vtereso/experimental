package fake

import (
	fakerouteclientset "github.com/openshift/client-go/route/clientset/versioned/fake"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/client"
	faketektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	faketriggerclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned/fake"
	fakek8sclientset "k8s.io/client-go/kubernetes/fake"
)

// DummyGroup returns a group using fake clients and defaults
func DummyGroup() *client.Group {
	return &client.Group{
		K8sClient:      dummyK8sClientset(),
		TektonClient:   dummyTektonClientset(),
		TriggersClient: dummyTriggersClientset(),
		RoutesClient:   dummyRoutesClientset(),
		Defaults:       dummyDefaults(),
	}
}

func dummyK8sClientset() *fakek8sclientset.Clientset {
	return fakek8sclientset.NewSimpleClientset()
}

func dummyTektonClientset() *faketektonclientset.Clientset {
	return faketektonclientset.NewSimpleClientset()
}

func dummyTriggersClientset() *faketriggerclientset.Clientset {
	return faketriggerclientset.NewSimpleClientset()
}

func dummyRoutesClientset() *fakerouteclientset.Clientset {
	return fakerouteclientset.NewSimpleClientset()
}

func dummyDefaults() client.EnvDefaults {
	return client.EnvDefaults{
		Namespace: "default",
		Platform:  "openshift",
	}
}
