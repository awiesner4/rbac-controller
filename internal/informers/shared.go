package informers

import (
	"os"
	"time"

	"github.com/awiesner4/rbac-controller/internal/kube"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

type CustomInformerFactory struct {
	Clientset kubernetes.Interface
	Factory   informers.SharedInformerFactory
}

func addInformers(customFactory *CustomInformerFactory) {
	roleBindingInformer(customFactory)
	// namespaceInformer(customFactory)
	// logrus.Info("RUN INFORMER LATER")
}

func NewCustomInformerFactory(clientset kubernetes.Interface, resync time.Duration) *CustomInformerFactory {
	// Configure the informer factory with the label selector
	optionsModifier := func(options *metav1.ListOptions) {
		kube.WithLabelSelector(options)
	}

	factory := informers.NewSharedInformerFactoryWithOptions(
		clientset,
		resync,
		informers.WithTweakListOptions(optionsModifier),
	)
	return &CustomInformerFactory{
		Clientset: clientset,
		Factory:   factory,
	}
}

func RunInformers(clientset kubernetes.Interface, stopCh <-chan struct{}) error {
	customFactory := NewCustomInformerFactory(clientset, 2*time.Minute)

	addInformers(customFactory)

	customFactory.Factory.Start(stopCh)
	maxRetries := 3
	synced := false

	for i := 0; i < maxRetries; i++ {
		logrus.Infof("Waiting for informer caches to sync (attempt %d/%d)", i+1, maxRetries)
		syncedMap := customFactory.Factory.WaitForCacheSync(stopCh)

		// Check that all informers have synced
		synced = true
		for informerType, success := range syncedMap {
			if !success {
				logrus.Warnf("Informer for %v failed to sync", informerType)
				synced = false
			}
		}

		if synced {
			logrus.Info("All informer caches synced successfully")
			break
		}

		logrus.Warn("Some informer caches failed to sync, retrying...")
		time.Sleep(2 * time.Second) // Delay between retries
	}

	if !synced {
		logrus.Error("Informer caches failed to sync after retries")
		os.Exit(1)
	}

	return nil
}
