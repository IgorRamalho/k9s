package dao

import (
	"context"
	"fmt"

	"github.com/derailed/k9s/internal"
	"github.com/derailed/k9s/internal/client"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

// Table retrieves K8s resources as tabular data.
type Table struct {
	Generic
}

// Get returns a given resource.
func (t *Table) Get(ctx context.Context, path string) (runtime.Object, error) {
	ns, n := client.Namespaced(path)

	a := fmt.Sprintf(gvFmt, metav1beta1.SchemeGroupVersion.Version, metav1beta1.GroupName)
	_, codec := t.codec()

	c, err := t.getClient()
	if err != nil {
		return nil, err
	}
	o, err := c.Get().
		SetHeader("Accept", a).
		Namespace(ns).
		Name(n).
		Resource(t.gvr.R()).
		VersionedParams(&metav1beta1.TableOptions{}, codec).
		Do().Get()
	if err != nil {
		return nil, err
	}

	return o, nil
}

// List all Resources in a given namespace.
func (t *Table) List(ctx context.Context, ns string) ([]runtime.Object, error) {
	labelSel, ok := ctx.Value(internal.KeyLabels).(string)
	if !ok {
		log.Debug().Msgf("No label selector found in context. Listing all resources")
	}

	a := fmt.Sprintf(gvFmt, metav1beta1.SchemeGroupVersion.Version, metav1beta1.GroupName)
	_, codec := t.codec()

	c, err := t.getClient()
	if err != nil {
		return nil, err
	}
	o, err := c.Get().
		SetHeader("Accept", a).
		Namespace(ns).
		Resource(t.gvr.R()).
		VersionedParams(&metav1.ListOptions{LabelSelector: labelSel}, codec).
		Do().Get()
	if err != nil {
		return nil, err
	}

	return []runtime.Object{o}, nil
}

// ----------------------------------------------------------------------------
// Helpers...

const gvFmt = "application/json;as=Table;v=%s;g=%s, application/json"

func (t *Table) getClient() (*rest.RESTClient, error) {
	crConfig := t.Client().RestConfigOrDie()
	gv := t.gvr.GV()
	crConfig.GroupVersion = &gv
	crConfig.APIPath = "/apis"
	if t.gvr.G() == "" {
		crConfig.APIPath = "/api"
	}
	codec, _ := t.codec()
	crConfig.NegotiatedSerializer = codec.WithoutConversion()

	crRestClient, err := rest.RESTClientFor(crConfig)
	if err != nil {
		return nil, err
	}
	return crRestClient, nil
}

func (t *Table) codec() (serializer.CodecFactory, runtime.ParameterCodec) {
	scheme := runtime.NewScheme()
	gv := t.gvr.GV()
	metav1.AddToGroupVersion(scheme, gv)
	scheme.AddKnownTypes(gv, &metav1beta1.Table{}, &metav1beta1.TableOptions{})
	scheme.AddKnownTypes(metav1beta1.SchemeGroupVersion, &metav1beta1.Table{}, &metav1beta1.TableOptions{})

	return serializer.NewCodecFactory(scheme), runtime.NewParameterCodec(scheme)
}
