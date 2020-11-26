package pkg

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	auth "github.com/deislabs/oras/pkg/auth/docker"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/opencontainers/go-digest"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog"
)

const (
	// SecretCfgPath should in JSON format, is the kubernetes.io/dockerconfigjson types of kubernetes secret.
	// The content includes your private docker registry FQDN, username, password, email.
	SecretCfgPath = "/etc/secret/dockerconfigjson"

	// WhiteListFilePath is the path of file whitelist.conf
	// The file content should in JSON format
	WhiteListFilePath = "/etc/config/whitelist.conf"

	// SchemeTypeHTTP defines const "http" for registry URL scheme
	SchemeTypeHTTP = "http"
	// SchemeTypeHTTPS defines const "https" for registry URL scheme
	SchemeTypeHTTPS = "https"

	// PrefixHTTP defines const SchemeTypeHTTP+"://"
	PrefixHTTP = SchemeTypeHTTP + "://"
	// PrefixHTTPS defines const SchemeTypeHTTPS+"://"
	PrefixHTTPS = SchemeTypeHTTPS + "://"

	// HelmChartConfigMediaType is the reserved media type for the Helm chart manifest config
	HelmChartConfigMediaType = "application/vnd.cncf.helm.config.v1+json"

	// HelmChartContentLayerMediaType is the reserved media type for Helm chart package content
	HelmChartContentLayerMediaType = "application/tar+gzip"
)

var (
	// GlobalBackend backend global var
	GlobalBackend *Backend

	// GlobalWhiteList store whitelist which read from file WhiteListFilePath
	GlobalWhiteList = &WhiteList{}

	// cache section
	refToChartCache = make(map[string]*HelmOCIConfig)

	pathToRefCache = make(map[string]RefData)

	l = sync.Mutex{}
)

// WhiteList defines the whitelist of harbor and registry
type WhiteList struct {
	Harbor   map[string][]string `json:"harbor"`
	Registry map[string][]string `json:"registry"`
}

// KnownMediaTypes give known media types
func KnownMediaTypes() []string {
	return []string{
		HelmChartConfigMediaType,
		HelmChartContentLayerMediaType,
	}
}

// HubOptions defines the options for oci registry
type HubOptions struct {
	Scheme   string // http or https
	URL      string
	Username string
	Password string
}

// DockerSecretCfg defines the structure of dockerconfigjson
type DockerSecretCfg struct {
	Auths map[string]SecretCfg `json:"auths"`
}

// SecretCfg defines the structure that in DockerSecretCfg struct
type SecretCfg struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Auth     string `json:"auth"`
}

// FullfillHubOptions get user info from SecretCfgPath.
// The o.scheme is used to connect to the registry.
func (o *HubOptions) FullfillHubOptions() error {
	klog.Infof("Try to load secret config file form %s", SecretCfgPath)
	body, err := ioutil.ReadFile(SecretCfgPath)
	if err != nil {
		klog.Warningf("Read file : %s failed, the reason is : %s", SecretCfgPath, err.Error())
	} else {
		var cfg DockerSecretCfg
		if err := json.Unmarshal(body, &cfg); err != nil {
			return err
		}

		for k, v := range cfg.Auths {
			if o.matchURL(k) {
				o.Username = v.Username
				o.Password = v.Password

				break
			}
		}
		klog.Infof("The secret config file was successfully loaded.")
	}

	return nil
}

// GetHost returns host DNS name
func (o *HubOptions) GetHost() string {
	host := o.URL
	uri, _ := url.Parse(o.URL)
	if uri != nil && uri.Host != "" {
		host = uri.Host
	}

	return host
}

// IsSchemeValid returns o.Scheme is http or https, or not
func (o *HubOptions) IsSchemeValid() bool {
	return o.Scheme == SchemeTypeHTTP || o.Scheme == SchemeTypeHTTPS
}

// ValidateAndSetScheme validate scheme from o.Scheme first.
// if o.Scheme is empty or other value, then get scheme from o.URL.
// if none of the above, infer the scheme.
func (o *HubOptions) ValidateAndSetScheme() {
	o.Scheme = strings.ToLower(o.Scheme)
	if o.IsSchemeValid() {
		return
	}

	if strings.HasPrefix(o.URL, PrefixHTTPS) {
		o.Scheme = SchemeTypeHTTPS
	} else if strings.HasPrefix(o.URL, PrefixHTTP) {
		o.Scheme = SchemeTypeHTTP
	} else {
		// Do nothing, need to try to infer
	}
}

func (o *HubOptions) matchURL(target string) bool {
	source := o.URL
	if strings.HasPrefix(source, PrefixHTTP) {
		source = source[len(PrefixHTTP):]
	} else if strings.HasPrefix(source, PrefixHTTPS) {
		source = source[len(PrefixHTTPS):]
	}

	if strings.HasPrefix(target, PrefixHTTP) {
		target = target[len(PrefixHTTP):]
	} else if strings.HasPrefix(target, PrefixHTTPS) {
		target = target[len(PrefixHTTPS):]
	}

	return source == target
}

// TryToNewRegistry first try to connect to the registry using https.
// If https fails, try to connect with http.
func (o *HubOptions) TryToNewRegistry() (*registry.Registry, error) {
	tryURL := fmt.Sprintf("%s%s", PrefixHTTPS, o.URL)
	klog.Infof("Try to connect to the registry using HTTPS scheme : %s.\n", tryURL)
	r, err := registry.NewInsecure(tryURL, o.Username, o.Password)
	if err != nil {
		klog.Warning("Failed to connect to the registry using HTTPS scheme.", err)

		tryURL := fmt.Sprintf("%s%s", PrefixHTTP, o.URL)
		klog.Infof("Try to connect to the registry using HTTP scheme : %s.\n", tryURL)
		r, err = registry.NewInsecure(tryURL, o.Username, o.Password)
		if err != nil {
			panic(err)
		} else {
			klog.Infof("Successfully connected to the registry : %s.", tryURL)
			o.URL = tryURL
			o.Scheme = SchemeTypeHTTP
		}
	} else {
		klog.Infof("Successfully connected to the registry : %s.", tryURL)
		o.URL = tryURL
		o.Scheme = SchemeTypeHTTPS
	}

	return r, nil
}

// NewResolver create a remotes.Resolver for pulling artifact
// see https://github.com/deislabs/oras/blob/master/cmd/oras/resolver.go
func (o *HubOptions) NewResolver(configs ...string) remotes.Resolver {
	plainHTTP := false
	if o.Scheme == SchemeTypeHTTP || strings.HasPrefix(o.URL, PrefixHTTP) {
		plainHTTP = true
	}

	opts := docker.ResolverOptions{
		PlainHTTP: plainHTTP,
	}

	opts.Client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	if o.Username != "" || o.Password != "" {
		opts.Credentials = func(hostName string) (string, string, error) {
			return o.Username, o.Password, nil
		}
		return docker.NewResolver(opts)
	}
	cli, err := auth.NewClient(configs...)
	if err != nil {
		klog.Warning("WARNING: Error loading auth file: ", err)
	}
	resolver, err := cli.Resolver(context.Background(), opts.Client, plainHTTP)
	if err != nil {
		klog.Warning("WARNING: Error loading resolver: ", err)
		resolver = docker.NewResolver(opts)
	}
	return resolver
}

// RefData defines the structure that contains the name and digist
type RefData struct {
	Name string
	//  digest of the data layer
	Digest digest.Digest
}

// HelmOCIConfig ... from oci manifest config
type HelmOCIConfig struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	APIVersion  string `json:"apiVersion"`
	AppVersion  string `json:"appVersion"`
	Type        string `json:"type"`

	// use first layer of content's now.
	//TODO: make sure this is ok
	Digest string `json:"-"`
}

// ToChartVersion convert HelmOCIConfig to repo.ChartVersion
func (h *HelmOCIConfig) ToChartVersion() *repo.ChartVersion {

	m := chart.Metadata{}
	m.Version = h.Version
	m.Name = h.Name
	m.APIVersion = h.APIVersion
	m.AppVersion = h.AppVersion
	m.Description = h.Description

	v := repo.ChartVersion{Metadata: &m}
	v.Digest = h.Digest
	v.URLs = []string{"charts/" + genPath(h.Name, h.Version)}

	return &v
}
