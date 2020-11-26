package pkg

import (
	"fmt"
	"strings"

	"github.com/containerd/containerd/remotes"
	"github.com/heroku/docker-registry-client/registry"
	"k8s.io/klog"
)

// Hub is an interface, can list []HelmOCIConfig from backend hub
type Hub interface {
	ListObjects() ([]HelmOCIConfig, error)
}

// Backend defines a Registry client or a Harbor client and the hub DNS address, and the remote resolver
type Backend struct {
	// Hub is backend of the remote hub
	Hub
	// Host is hub DNS address
	Host string
	// Resolver provides remotes based on a locator. It can fetches content from remote hub
	Resolver remotes.Resolver
}

// NewBackend create a Backend
// Try to create Harbor backend first, if it fails, continue to create Registry backend
func NewBackend(opts *HubOptions) *Backend {
	if b, ok := newHarborBackend(opts); ok {
		return b
	}

	return newRegistryBackend(opts)
}

func newHarborBackend(opts *HubOptions) (*Backend, bool) {
	hc := NewHarborClient(opts.URL, opts.Username, opts.Password)
	if !hc.ValidateHarborV2() {
		klog.Warningf("%s is not harbor v2", opts.URL)
		return nil, false
	}

	klog.Infof("Created a harbor v2 type backend, which will connect to %s", opts.URL)
	return &Backend{
		Host:     opts.GetHost(),
		Resolver: opts.NewResolver(),
		Hub: &HarborHub{
			Harbor: hc,
		},
	}, true
}

// NewRegistryBackend create a Registry client and return a Backend structure
func newRegistryBackend(opts *HubOptions) *Backend {
	var reg *registry.Registry
	var err error
	opts.ValidateAndSetScheme()

	if !opts.IsSchemeValid() {
		reg, err = opts.TryToNewRegistry()
		if err != nil {
			panic(err)
		}
	} else {
		prefix := opts.Scheme + "://"
		if !strings.HasPrefix(opts.URL, prefix) {
			opts.URL = fmt.Sprintf("%s%s", prefix, opts.URL)
		}

		reg, err = registry.NewInsecure(opts.URL, opts.Username, opts.Password)
		if err != nil {
			panic(err)
		}
	}

	klog.Infof("Created a docker-registry type backend, which will connect to %s", opts.URL)
	return &Backend{
		Host:     opts.GetHost(),
		Resolver: opts.NewResolver(),
		Hub: &RegistryHub{
			Registry: reg,
		},
	}
}
