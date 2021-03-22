package utils

import (
	"flag"
	jx "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	largetestv1beta1 "github.com/vitech-team/sdlcctl/apis/largetest/v1beta1"
	sdlc "github.com/vitech-team/sdlcctl/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
)

//var kubeconfig *string
//
//func init() {
//	if home := homedir.HomeDir(); home != "" {
//		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
//	} else {
//		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
//	}
//}

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = largetestv1beta1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func NewLazyClients(kubeClient kubernetes.Interface, jxClient jx.Interface, sdlcClient sdlc.Interface) (kubernetes.Interface, jx.Interface, sdlc.Interface) {

	if kubeClient != nil {
		return kubeClient, jxClient, sdlcClient
	}

	if !flag.Parsed() {
		flag.Parse()
	}

	config := ctrl.GetConfigOrDie()

	// create the clientset
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	sdlcClient, err = sdlc.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	jxClient, err = jx.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return kubeClient, jxClient, sdlcClient
}
