package pkg

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

const (
	// GroupName is CRD GroupName
	GroupName = "product.alauda.io"
	// GroupVersion is CRD GroupVersion
	GroupVersion = "v1alpha1"
	// ProductBases is CRD resource
	ProductBases = "productbases"
)

var (
	// SchemeGroupVersion set CRD SchemeGroupVersion
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}
	// SchemeBuilder new SchemeBuilder by addKnownTypes
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme is SchemeBuilder.AddToScheme
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&ProductBase{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

// getClient init a RESTClient in kubernetes cluster
func getClient() (*rest.RESTClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	config.GroupVersion = &SchemeGroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	AddToScheme(scheme.Scheme)

	clientSet, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}

	return clientSet, nil
}

func getProductBase(name string) (*ProductBase, error) {
	var pb = ProductBase{}
	var cli *rest.RESTClient
	var err error
	if cli, err = getClient(); err != nil {
		return nil, err
	}

	err = cli.Get().Resource(ProductBases).
		Name(name).
		VersionedParams(&metav1.GetOptions{}, scheme.ParameterCodec).
		Do(context.Background()).
		Into(&pb)

	if err != nil {
		return nil, err
	}

	return &pb, nil
}
