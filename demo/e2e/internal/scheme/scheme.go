package scheme

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	demomctestiov1alpha1 "github.com/filariow/mctest/demo/show/api/v1alpha1"
)

var (
	DefaultSchemeHost *runtime.Scheme = scheme.Scheme
)

func init() {
	utilruntime.Must(demomctestiov1alpha1.AddToScheme(DefaultSchemeHost))
}
