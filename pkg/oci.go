package pkg

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
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

// ==================================
// ListObjects parser all helm chart basic info from oci manifest
// skip all manifests that are not helm type
func (b *Backend) ListObjects2() ([]HelmOCIConfig, error) {
	if b.Harbor != nil {
		return b.listObjectsFromHarbor2()
	}

	return b.listObjectsFromRegistry()
}

func (b *Backend) listObjectsFromRegistry() ([]HelmOCIConfig, error) {
	repositories, err := b.Hub.Repositories()
	if err != nil {
		return nil, err
	}

	objects := make([]HelmOCIConfig, 0)
	var isBreak bool
	if len(GlobalWhiteList.Registry) > 0 {
		imagesTagsMap := matchImagesForRegistry(repositories)

		for image, tags := range imagesTagsMap {
			if len(tags) > 0 {
				for _, tag := range tags {
					objects, isBreak, err = b.listObjectsByImageAndTag(image, tag, objects)
					if err != nil {
						if isBreak {
							klog.Error("err lsit objects by image and tag", err)
							break
						}

						klog.Warning("err lsit objects by image and tag", err)
					}
				}
			} else {
				objects, isBreak, err = b.listObjectsByImage(image, objects)
				if err != nil {
					if isBreak {
						klog.Error("err lsit objects by image", err)
						break
					}

					klog.Warning("err lsit objects by image", err)
				}
			}
		}
	} else {
		for _, image := range repositories {
			objects, isBreak, err = b.listObjectsByImage(image, objects)
			if err != nil {
				if isBreak {
					klog.Error("err lsit objects by image", err)
					break
				}

				klog.Warning("err lsit objects by image", err)
			}
		}
	}

	return objects, nil
}

// matchImagesForRegistry match regexp of images, the regexp string read from whitelist
func matchImagesForRegistry(repositories []string) map[string][]string {
	ret := make(map[string][]string)
	for _, originalImage := range repositories {
		for regexpImage, tags := range GlobalWhiteList.Registry {
			r, _ := regexp.Compile(regexpImage)
			if r != nil && r.MatchString(originalImage) {
				ret[originalImage] = tags
				// The match ends once image has been matched
				break
			}
		}
	}

	return ret
}

// listObjectsByImage return []HelmOCIConfig, isBreak, error
func (b *Backend) listObjectsByImage(image string, objects []HelmOCIConfig) ([]HelmOCIConfig, bool, error) {
	tags, err := b.Hub.Tags(image)
	if err != nil {
		klog.Error("err list tags for repo: ", err)
		// You can list Repositories, but the API returns UNAUTHORIZED or PROJECT_POLICY_VIOLATION when you list tags for a repository
		if strings.Contains(err.Error(), "repository name not known to registry") ||
			strings.Contains(err.Error(), "UNAUTHORIZED") ||
			strings.Contains(err.Error(), "PROJECT_POLICY_VIOLATION") {

			// need break
			return objects, false, err
		}

		return objects, true, err
	}

	for _, tag := range tags {
		var isBreak bool
		var err error
		objects, isBreak, err = b.listObjectsByImageAndTag(image, tag, objects)
		if err != nil {
			if isBreak {
				klog.Error("err lsit objects by image and tag", err)
				// need break
				return objects, true, err
			}

			klog.Warning("err lsit objects by image and tag", err)
		}
	}

	return objects, false, nil
}

// listObjectsByImageAndTag return []HelmOCIConfig, isBreak, error
func (b *Backend) listObjectsByImageAndTag(image, tag string, objects []HelmOCIConfig) ([]HelmOCIConfig, bool, error) {
	manifest, err := b.Hub.OCIManifestV1(image, tag)
	if err != nil {
		klog.Warning("err get manifest for tag: ", err)
		// You can list tags, but the API returns UNAUTHORIZED or PROJECT_POLICY_VIOLATION when you get manifest for a tag
		if strings.Contains(err.Error(), "UNAUTHORIZED") ||
			strings.Contains(err.Error(), "PROJECT_POLICY_VIOLATION") {
			return objects, true, err
		}

		// FIXME: continue or break error.
		return objects, false, err
	}

	// if one tag is not helm, consider this image is not
	if manifest.Config.MediaType != HelmChartConfigMediaType {
		return objects, true, err
	}

	// only one layer is allowed
	if len(manifest.Layers) != 1 {
		return objects, true, err
	}

	ref := image + ":" + tag

	// lookup in cache first
	obj := refToChartCache[ref]
	if obj != nil {
		objects = append(objects, *obj)
		return objects, false, nil
	}

	// fetch manifest config and parse to helm info
	digest := manifest.Config.Digest
	result, err := b.Hub.DownloadBlob(image, digest)
	if err != nil {
		return objects, true, err
	}
	body, err := ioutil.ReadAll(result)
	if err != nil {
		return objects, true, err
	}
	result.Close()

	cfg := &HelmOCIConfig{}
	err = json.Unmarshal(body, cfg)
	if err != nil {
		return objects, true, err
	}

	cfg.Digest = manifest.Layers[0].Digest.Encoded()
	objects = append(objects, *cfg)

	// may be helm and captain are pulling same time
	l.Lock()
	refToChartCache[ref] = cfg
	pathToRefCache[genPath(cfg.Name, cfg.Version)] = RefData{
		Name:   ref,
		Digest: manifest.Layers[0].Digest,
	}
	l.Unlock()

	return objects, false, nil
}

func (b *Backend) listObjectsFromHarbor2() ([]HelmOCIConfig, error) {
	objects := make([]HelmOCIConfig, 0)
	if len(GlobalWhiteList.Harbor) > 0 {
		for p, rs := range GlobalWhiteList.Harbor {
			if len(rs) > 0 {
				for _, r := range rs {
					objects, _ = b.listArtifactsByRepository(p, r, objects)
				}
			} else {
				objects, _ = b.listArtifactsByProject(p, objects)
			}
		}
	} else {
		projects, err := b.Harbor2.ListProjects()
		if err != nil {
			return nil, err
		}

		for _, p := range projects {
			objects, _ = b.listArtifactsByProject(p.Name, objects)
		}
	}

	return objects, nil
}

func (b *Backend) listArtifactsByProject(projectName string, objects []HelmOCIConfig) ([]HelmOCIConfig, error) {
	repositories, err := b.Harbor2.ListRepositories(projectName)
	if err != nil {
		klog.Warning("List repositories error", err)
		return objects, err
	}

	for _, repo := range repositories {
		_name := repo.Name
		if strings.HasPrefix(_name, projectName) {
			_name = _name[len(projectName)+1:]
		}

		objects, _ = b.listArtifactsByRepository(projectName, _name, objects)
	}

	return objects, nil
}

func (b *Backend) listArtifactsByRepository(projectName, repositoryName string, objects []HelmOCIConfig) ([]HelmOCIConfig, error) {
	artifacts, err := b.Harbor2.ListArtifacts(projectName, repositoryName)
	if err != nil {
		klog.Warning("List artifacts error", err)
		return objects, err
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
		ref := fmt.Sprintf("%s/%s:%s", projectName, repositoryName, cfg.Version)
		digest, _ := digest.Parse(cfg.Digest)
		l.Lock()
		pathToRefCache[genPath(cfg.Name, cfg.Version)] = RefData{
			Name:   ref,
			Digest: digest,
		}
		l.Unlock()
	}

	return objects, nil
}
