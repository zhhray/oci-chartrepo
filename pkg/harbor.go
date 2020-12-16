package pkg

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/opencontainers/go-digest"
	"k8s.io/klog"
)

var projMap = map[string]string{}
var repoMap = map[string]string{}

// BasicAuth provides basic http authentication to a request passed via context using basic
type BasicAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// HarborClient provides a BasePath, a http client and a BasicAuth
type HarborClient struct {
	BasePath   string
	HTTPClient *http.Client
	Auth       *BasicAuth
}

// NewHarborClient return a HarborClient instance
func NewHarborClient(urlStr, username, password string) *HarborClient {
	if !strings.HasPrefix(urlStr, PrefixHTTPS) {
		urlStr = fmt.Sprintf("%s%s", PrefixHTTPS, urlStr)
	}

	return &HarborClient{
		BasePath: strings.TrimSuffix(urlStr, "/") + "/api/v2.0",
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			},
		},
		Auth: &BasicAuth{
			Username: username,
			Password: password,
		},
	}
}

func (hc *HarborClient) url(pathTemplate string, args ...interface{}) string {
	pathSuffix := fmt.Sprintf(pathTemplate, args...)
	return fmt.Sprintf("%s%s", hc.BasePath, pathSuffix)
}

// ValidateHarborV2 validate remote registy type is harbor v2 or not
func (hc *HarborClient) ValidateHarborV2() bool {
	urlStr := hc.url("/systeminfo")
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		klog.Warningf("http new request error %s", err.Error())
		return false
	}
	req.SetBasicAuth(hc.Auth.Username, hc.Auth.Password)
	klog.Infof("get systeminfo url = %s", urlStr)
	resp, err := hc.HTTPClient.Do(req)
	if err != nil {
		klog.Warningf("http client do request error %s", err.Error())
		return false
	}
	defer resp.Body.Close()

	encoder := json.NewDecoder(resp.Body)
	generalInfo := &GeneralInfo{}
	err = encoder.Decode(&generalInfo)
	if err != nil {
		klog.Warningf("json decode generalInfo error %s", err.Error())
		return false
	}

	klog.Infof("systeminfo is %+v", generalInfo)
	return strings.HasPrefix(generalInfo.HarborVersion, "v2")
}

// GeneralInfo the struct of harbor GeneralInfo
type GeneralInfo struct {
	// The build version of Harbor.
	HarborVersion string `json:"harbor_version,omitempty"`
}

// ListProjects list projects from a harbor with v2 api
func (hc *HarborClient) ListProjects() ([]*Project, error) {
	urlStr := hc.url("/projects?page_size=500")
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(hc.Auth.Username, hc.Auth.Password)
	klog.Infof("list projects url = %s", urlStr)
	resp, err := hc.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	projects := []*Project{}
	if resp.StatusCode < 300 {
		encoder := json.NewDecoder(resp.Body)
		err = encoder.Decode(&projects)
		if err != nil {
			return nil, err
		}
	}

	return projects, nil
}

// Project the struct of harbor Project
type Project struct {
	// Project ID
	ProjectID int32 `json:"project_id,omitempty"`
	// The name of the project.
	Name string `json:"name,omitempty"`
}

// ListRepositories list repositories from a harbor with v2 api
func (hc *HarborClient) ListRepositories(projectName string, name string) ([]*Repository, error) {
	urlStr := hc.url("/projects/%s/repositories?page_size=500", projectName)
	if name != "" {
		// harbor api supports fuzzy matching of name with '~'
		query := fmt.Sprintf("name=~%s", name)
		urlStr = fmt.Sprintf("%s&q=%s", urlStr, url.QueryEscape(query))
	}
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(hc.Auth.Username, hc.Auth.Password)
	klog.Infof("list repositories url = %s", urlStr)
	resp, err := hc.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	repositories := []*Repository{}
	if resp.StatusCode < 300 {
		encoder := json.NewDecoder(resp.Body)
		err = encoder.Decode(&repositories)
		if err != nil {
			return nil, err
		}
	}

	return repositories, nil
}

// Repository the struct of harbor Repository
type Repository struct {
	// The ID of the repository
	ID int64 `json:"id,omitempty"`
	// The ID of the project that the repository belongs to
	ProjectID int64 `json:"project_id,omitempty"`
	// The name of the repository
	Name string `json:"name,omitempty"`
}

// ListArtifacts list artifacts from a harbor with v2 api
func (hc *HarborClient) ListArtifacts(projectName, repoName string) ([]*Artifact, error) {
	query := fmt.Sprintf("q=media_type=%s", HelmChartConfigMediaType)
	urlStr := hc.url("/projects/%s/repositories/%s/artifacts?page_size=500&%s", projectName, repoName, url.QueryEscape(query))
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(hc.Auth.Username, hc.Auth.Password)
	klog.Infof("list artifacts url = %s", urlStr)
	resp, err := hc.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	artifacts := []*Artifact{}
	if resp.StatusCode < 300 {
		encoder := json.NewDecoder(resp.Body)
		err = encoder.Decode(&artifacts)
		if err != nil {
			return nil, err
		}
	}

	return artifacts, nil
}

// GetArtifact get an artifact from a harbor with v2 api
func (hc *HarborClient) GetArtifact(projectName, repositoryName, tag string) (*Artifact, error) {
	urlStr := hc.url("/projects/%s/repositories/%s/artifacts/%s", projectName, repositoryName, tag)
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(hc.Auth.Username, hc.Auth.Password)
	klog.Infof("get artifact url = %s", urlStr)
	resp, err := hc.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	artifact := &Artifact{}
	if resp.StatusCode < 300 {
		encoder := json.NewDecoder(resp.Body)
		err = encoder.Decode(&artifact)
		if err != nil {
			return nil, err
		}
	}

	return artifact, nil
}

// Artifact the struct of harbor Artifact
type Artifact struct {
	// The ID of the artifact
	ID int64 `json:"id,omitempty"`
	// The media type of the artifact
	MediaType string `json:"media_type,omitempty"`
	// The manifest media type of the artifact
	ManifestMediaType string `json:"manifest_media_type,omitempty"`
	// The ID of the project that the artifact belongs to
	ProjectID int64 `json:"project_id,omitempty"`
	// The ID of the repository that the artifact belongs to
	RepositoryID int64 `json:"repository_id,omitempty"`
	// The digest of the artifact
	Digest string `json:"digest,omitempty"`
	// The size of the artifact
	Size int64 `json:"size,omitempty"`
	// ExtraAttrs  extra attrs of the artiact
	ExtraAttrs map[string]interface{} `json:"extra_attrs,omitempty"`
}

// HarborHub is harbor backend will implement Hub interface
type HarborHub struct {
	Harbor *HarborClient
}

// ListObjects parser all helm chart basic info from oci manifest
// skip all manifests that are not helm type
func (h *HarborHub) ListObjects() ([]HelmOCIConfig, error) {
	st := time.Now()
	objects := loadObjectsFromCache()

	if GlobalWhiteList.Mode == StrictMode {
		for _, cv := range GlobalWhiteList.ChartVersions {
			// cv.Name: acp/chart-alauda-container-platform
			ss := strings.Split(cv.Name, "/")
			if len(ss) == 2 {
				p := ss[0]
				r := ss[1]

				objects, _ = h.listArtifactsByRepositoryAndTag(p, r, cv.Version, objects)
			}
		}
	} else {
		if len(GlobalWhiteList.Harbor) > 0 {
			for p, rs := range GlobalWhiteList.Harbor {
				if len(rs) > 0 {
					repos := []*Repository{}
					for _, r := range rs {
						rets, err := h.Harbor.ListRepositories(p, r)
						if err != nil {
							continue
						}

						for _, ret := range rets {
							repos = append(repos, ret)
						}
					}

					for _, repo := range repos {
						objects, _ = h.listArtifactsByRepository(p, repo.Name, objects)
					}
				} else {
					objects, _ = h.listArtifactsByProject(p, objects)
				}

				projMap[p] = p
			}
		} else {
			projects, err := h.Harbor.ListProjects()
			if err != nil {
				return nil, err
			}

			for _, p := range projects {
				objects, _ = h.listArtifactsByProject(p.Name, objects)
				projMap[p.Name] = p.Name
			}
		}
	}

	klog.Infof("======proj length is %d", len(projMap))
	klog.Infof("======repo length is %d", len(repoMap))
	klog.Infof("======objects length is %d", len(objects))
	ed := time.Now()
	sub := ed.Sub(st)
	klog.Infof("======listObjects use %f min", sub.Minutes())

	return objects, nil
}

func (h *HarborHub) listArtifactsByProject(projectName string, objects []HelmOCIConfig) ([]HelmOCIConfig, error) {
	repositories, err := h.Harbor.ListRepositories(projectName, "")
	if err != nil {
		klog.Warning("List repositories error", err)
		return objects, err
	}

	for _, repo := range repositories {
		objects, _ = h.listArtifactsByRepository(projectName, repo.Name, objects)
	}

	return objects, nil
}

func (h *HarborHub) listArtifactsByRepository(projectName, repositoryName string, objects []HelmOCIConfig) ([]HelmOCIConfig, error) {
	if strings.HasPrefix(repositoryName, projectName) {
		repositoryName = repositoryName[len(projectName)+1:]
	}
	artifacts, err := h.Harbor.ListArtifacts(projectName, repositoryName)
	if err != nil {
		klog.Warning("List artifacts error", err)
		return objects, err
	}

	for _, atf := range artifacts {
		ref := fmt.Sprintf("%s/%s", projectName, repositoryName)
		objects, _ = convertArtifactIntoObjects(atf, ref, objects)
	}

	return objects, nil
}

func (h *HarborHub) listArtifactsByRepositoryAndTag(projectName, repositoryName, tag string, objects []HelmOCIConfig) ([]HelmOCIConfig, error) {
	atf, err := h.Harbor.GetArtifact(projectName, repositoryName, tag)
	if err != nil {
		klog.Warning("Get artifact error", err)
		return objects, err
	}

	ref := fmt.Sprintf("%s/%s", projectName, repositoryName)
	objects, _ = convertArtifactIntoObjects(atf, ref, objects)

	return objects, nil
}

func convertArtifactIntoObjects(atf *Artifact, ref string, objects []HelmOCIConfig) ([]HelmOCIConfig, error) {
	if atf == nil {
		return objects, nil
	}

	if atf.MediaType != HelmChartConfigMediaType {
		return objects, nil
	}

	if objectAlreadyExist(objects, atf.Digest) {
		return objects, nil
	}

	body, err := json.Marshal(atf.ExtraAttrs)
	if err != nil {
		klog.Warning("Json Marshal artifcat extra_attrs error", err)
		return objects, nil
	}

	cfg := &HelmOCIConfig{}
	if err := json.Unmarshal(body, cfg); err != nil {
		klog.Warning("Json Unmarshal artifcat extra_attrs to HelmOCIConfig error", err)
		return objects, nil
	}

	cfg.Digest = atf.Digest
	objects = append(objects, *cfg)

	digest, err := digest.Parse(cfg.Digest)
	if err != nil {
		klog.Warning("cfg.Digest Parse error", err)
		return objects, err
	}

	ref = fmt.Sprintf("%s:%s", ref, cfg.Version)
	// may be helm and captain are pulling same time
	l.Lock()
	refToChartCache[atf.Digest] = cfg
	pathToRefCache[genPath(cfg.Name, cfg.Version)] = RefData{
		Name:   ref,
		Digest: digest,
	}
	l.Unlock()

	return objects, nil
}
