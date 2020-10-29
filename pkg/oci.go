package pkg

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	harborV2 "github.com/alauda/oci-chartrepo/pkg/harbor/v2"
	"github.com/containerd/containerd/remotes"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/opencontainers/go-digest"
	"k8s.io/klog"
)

var (

	// GlobalBackend backend global var
	GlobalBackend *Backend

	// cache section
	refToChartCache map[string]*HelmOCIConfig

	pathToRefCache map[string]RefData
)

var l = sync.Mutex{}

func init() {
	refToChartCache = make(map[string]*HelmOCIConfig)
	pathToRefCache = make(map[string]RefData)
}

// Backend defines a RUL address and a Registry client
type Backend struct {
	URL      string
	Hub      *registry.Registry
	Harbor   *HarborClient
	Harbor2  *HarborClient2
	IsHarbor bool
	Host     string // registry DNS name
	Resolver remotes.Resolver
}

// HarborClient ...
type HarborClient struct {
	APIClient   *harborV2.APIClient
	AuthContext context.Context
}

// NewBackend create a remote registry client and return a Backend structure
func NewBackend(opts *RegistryOptions) *Backend {
	if opts.IsHarbor {
		// return newHarborBackend(opts)
		return newHarborBackend2(opts)
	}

	return newRegistryBackend(opts)
}

func newHarborBackend(opts *RegistryOptions) *Backend {
	if !strings.HasPrefix(opts.URL, "https://") {
		opts.URL = fmt.Sprintf("https://%s", opts.URL)
	}

	cfg := harborV2.NewConfiguration()
	cfg.BasePath = opts.URL + "/api/v2.0"
	cfg.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec
			},
		},
	}

	hc := &HarborClient{
		APIClient: harborV2.NewAPIClient(cfg),
		AuthContext: context.WithValue(context.Background(), harborV2.ContextBasicAuth, harborV2.BasicAuth{
			UserName: opts.Username,
			Password: opts.Password,
		}),
	}

	klog.Infof("Created a harbor type backend, which will connect to %s", opts.URL)
	return &Backend{
		URL:      cfg.BasePath,
		Harbor:   hc,
		IsHarbor: true,
		Host:     opts.GetHost(),
		Resolver: opts.NewResolver(),
	}
}

func newHarborBackend2(opts *RegistryOptions) *Backend {
	if !strings.HasPrefix(opts.URL, "https://") {
		opts.URL = fmt.Sprintf("https://%s", opts.URL)
	}

	hc2 := NewHarborClient2(opts.URL, opts.Username, opts.Password)
	if !hc2.ValidateHarborV2() {
		panic(errors.New(fmt.Sprintf("%s is not harbor v2", opts.URL)))
	}

	klog.Infof("Created a harbor type backend, which will connect to %s", opts.URL)
	return &Backend{
		URL:      hc2.BasePath,
		Harbor2:  hc2,
		IsHarbor: true,
		Host:     opts.GetHost(),
		Resolver: opts.NewResolver(),
	}
}

// NewRegistryBackend create a Registry client and return a Backend structure
func newRegistryBackend(opts *RegistryOptions) *Backend {
	var hub *registry.Registry
	var err error
	opts.ValidateAndSetScheme()

	if !opts.IsSchemeValid() {
		hub, err = opts.TryToNewRegistry()
		if err != nil {
			panic(err)
		}
	} else {
		prefix := opts.Scheme + "://"
		if !strings.HasPrefix(opts.URL, prefix) {
			opts.URL = fmt.Sprintf("%s%s", prefix, opts.URL)
		}

		hub, err = registry.NewInsecure(opts.URL, opts.Username, opts.Password)
		if err != nil {
			panic(err)
		}
	}

	klog.Infof("Created a docker-registry type backend, which will connect to %s", opts.URL)
	return &Backend{
		URL:      opts.URL,
		Hub:      hub,
		IsHarbor: false,
	}
}

// ListObjects parser all helm chart basic info from oci manifest
// skip all manifests that are not helm type
func (b *Backend) ListObjects() ([]HelmOCIConfig, error) {
	if b.IsHarbor {
		return b.listObjectsFromHarbor2()
	}

	repositories, err := b.Hub.Repositories()
	if err != nil {
		return nil, err
	}

	var objects []HelmOCIConfig

	for _, image := range repositories {
		tags, err := b.Hub.Tags(image)
		if err != nil {
			klog.Error("err list tags for repo: ", err)
			// You can list Repositories, but the API returns UNAUTHORIZED or PROJECT_POLICY_VIOLATION when you list tags for a repository
			if strings.Contains(err.Error(), "repository name not known to registry") ||
				strings.Contains(err.Error(), "UNAUTHORIZED") ||
				strings.Contains(err.Error(), "PROJECT_POLICY_VIOLATION") {

				continue
			}
			return nil, err
		}
		for _, tag := range tags {
			manifest, err := b.Hub.OCIManifestV1(image, tag)
			if err != nil {
				klog.Warning("err get manifest for tag: ", err)
				// You can list tags, but the API returns UNAUTHORIZED or PROJECT_POLICY_VIOLATION when you get manifest for a tag
				if strings.Contains(err.Error(), "UNAUTHORIZED") ||
					strings.Contains(err.Error(), "PROJECT_POLICY_VIOLATION") {
					break
				}

				// FIXME: continue or return error.
				continue
			}

			// if one tag is not helm, consider this image is not
			if manifest.Config.MediaType != HelmChartConfigMediaType {
				break
			}

			// only one layer is allowed
			if len(manifest.Layers) != 1 {
				break
			}

			ref := image + ":" + tag

			// lookup in cache first
			obj := refToChartCache[ref]
			if obj != nil {
				objects = append(objects, *obj)
				continue
			}

			// fetch manifest config and parse to helm info
			digest := manifest.Config.Digest
			result, err := b.Hub.DownloadBlob(image, digest)
			if err != nil {
				return nil, err
			}
			body, err := ioutil.ReadAll(result)
			if err != nil {
				return nil, err
			}
			result.Close()

			cfg := &HelmOCIConfig{}
			err = json.Unmarshal(body, cfg)
			if err != nil {
				return nil, err
			}

			cfg.Digest = manifest.Layers[0].Digest.Encoded()
			objects = append(objects, *cfg)

			// may be helm and captain are pulling same time
			l.Lock()
			refToChartCache[ref] = cfg
			pathToRefCache[genPath(cfg.Name, cfg.Version)] = RefData{
				Name:   image,
				Digest: manifest.Layers[0].Digest,
			}
			l.Unlock()
		}

	}
	return objects, nil

}

func (b *Backend) listObjectsFromHarbor() ([]HelmOCIConfig, error) {
	projects, _, err := b.Harbor.APIClient.ProjectApi.ListProjects(b.Harbor.AuthContext, &harborV2.ProjectApiListProjectsOpts{})
	if err != nil {
		return nil, err
	}

	var objects []HelmOCIConfig
	for _, p := range projects {
		repositories, _, err := b.Harbor.APIClient.RepositoryApi.ListRepositories(b.Harbor.AuthContext, p.Name, &harborV2.RepositoryApiListRepositoriesOpts{})
		if err != nil {
			klog.Warning("List repositories error", err)
			continue
		}

		for _, repo := range repositories {
			_name := repo.Name
			if strings.HasPrefix(_name, p.Name) {
				_name = _name[len(p.Name)+1:]
			}
			artifacts, _, err := b.Harbor.APIClient.ArtifactApi.ListArtifacts(b.Harbor.AuthContext, p.Name, _name, &harborV2.ArtifactApiListArtifactsOpts{})
			if err != nil {
				klog.Warning("List artifacts error", err)
				continue
			}

			for _, atf := range artifacts {
				if atf.MediaType != HelmChartConfigMediaType {
					continue
				}

				body, err := json.Marshal(atf.ExtraAttrs)
				if err != nil {
					klog.Warning("Json Marshal artifcat extra_attrs error", err)
					continue
				}

				cfg := &HelmOCIConfig{}
				if err := json.Unmarshal(body, cfg); err != nil {
					klog.Warning("Json Unmarshal artifcat extra_attrs to HelmOCIConfig error", err)
					continue
				}

				cfg.Digest = atf.Digest
				objects = append(objects, *cfg)

				// put into cache
				ref := repo.Name + ":" + cfg.Version
				l.Lock()
				pathToRefCache[genPath(cfg.Name, cfg.Version)] = RefData{
					Name:   ref,
					Digest: digest.FromString(cfg.Digest),
				}
				l.Unlock()
			}
		}
	}

	return objects, nil
}

func (b *Backend) listObjectsFromHarbor2() ([]HelmOCIConfig, error) {
	projects, err := b.Harbor2.ListProjects()
	if err != nil {
		return nil, err
	}

	var objects []HelmOCIConfig
	for _, p := range projects {
		repositories, err := b.Harbor2.ListRepositories(p.Name)
		if err != nil {
			klog.Warning("List repositories error", err)
			continue
		}

		for _, repo := range repositories {
			_name := repo.Name
			if strings.HasPrefix(_name, p.Name) {
				_name = _name[len(p.Name)+1:]
			}
			artifacts, err := b.Harbor2.ListArtifacts(p.Name, _name)
			if err != nil {
				klog.Warning("List artifacts error", err)
				continue
			}

			for _, atf := range artifacts {
				if atf.MediaType != HelmChartConfigMediaType {
					continue
				}

				body, err := json.Marshal(atf.ExtraAttrs)
				if err != nil {
					klog.Warning("Json Marshal artifcat extra_attrs error", err)
					continue
				}

				cfg := &HelmOCIConfig{}
				if err := json.Unmarshal(body, cfg); err != nil {
					klog.Warning("Json Unmarshal artifcat extra_attrs to HelmOCIConfig error", err)
					continue
				}

				cfg.Digest = atf.Digest
				objects = append(objects, *cfg)

				// put into cache
				ref := repo.Name + ":" + cfg.Version
				l.Lock()
				pathToRefCache[genPath(cfg.Name, cfg.Version)] = RefData{
					Name:   ref,
					Digest: digest.FromString(cfg.Digest),
				}
				l.Unlock()
			}
		}
	}

	return objects, nil
}
