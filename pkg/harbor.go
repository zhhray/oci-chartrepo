package pkg

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"k8s.io/klog"
)

// BasicAuth provides basic http authentication to a request passed via context using basic
type BasicAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// HarborClient2 provides a http client and a BasicAuth
type HarborClient2 struct {
	BasePath   string
	HTTPClient *http.Client
	Auth       *BasicAuth
}

// NewHarborClient2 return a HarborClient2 instence
func NewHarborClient2(url, username, password string) *HarborClient2 {
	return &HarborClient2{
		BasePath: url + "/api/v2.0",
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

// ValidateHarborV2 validate remote registy type is harbor v2 or not
func (hc *HarborClient2) ValidateHarborV2() bool {
	url := fmt.Sprintf("%s/systeminfo", hc.BasePath)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		klog.Warningf("http new request error %s", err.Error())
		return false
	}
	req.SetBasicAuth(hc.Auth.Username, hc.Auth.Password)
	klog.Infof("get systeminfo url = %s", url)
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
	// If the Harbor instance is deployed with nested notary.
	WithNotary bool `json:"with_notary,omitempty"`
	// If the Harbor instance is deployed with nested chartmuseum.
	WithChartmuseum bool `json:"with_chartmuseum,omitempty"`
	// The url of registry against which the docker command should be issued.
	RegistryURL string `json:"registry_url,omitempty"`
	// The external URL of Harbor, with protocol.
	ExternalURL string `json:"external_url,omitempty"`
	// The auth mode of current Harbor instance.
	AuthMode string `json:"auth_mode,omitempty"`
	// Indicate who can create projects, it could be 'adminonly' or 'everyone'.
	ProjectCreationRestriction string `json:"project_creation_restriction,omitempty"`
	// Indicate whether the Harbor instance enable user to register himself.
	SelfRegistration bool `json:"self_registration,omitempty"`
	// Indicate whether there is a ca root cert file ready for download in the file system.
	HasCaRoot bool `json:"has_ca_root,omitempty"`
	// The build version of Harbor.
	HarborVersion string `json:"harbor_version,omitempty"`
}

// ListProjects list projects from a harbor with v2 api
func (hc *HarborClient2) ListProjects() ([]*Project, error) {
	url := fmt.Sprintf("%s/projects", hc.BasePath)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(hc.Auth.Username, hc.Auth.Password)
	klog.Infof("list projects url = %s", url)
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
	// The owner ID of the project always means the creator of the project.
	OwnerID int32 `json:"owner_id,omitempty"`
	// The name of the project.
	Name string `json:"name,omitempty"`
	// The ID of referenced registry when the project is a proxy cache project.
	RegistryID int64 `json:"registry_id,omitempty"`
	// The creation time of the project.
	CreationTime time.Time `json:"creation_time,omitempty"`
	// The update time of the project.
	UpdateTime time.Time `json:"update_time,omitempty"`
	// A deletion mark of the project.
	Deleted bool `json:"deleted,omitempty"`
	// The owner name of the project.
	OwnerName string `json:"owner_name,omitempty"`
	// Correspond to the UI about whether the project's publicity is  updatable (for UI)
	Togglable bool `json:"togglable,omitempty"`
	// The role ID with highest permission of the current user who triggered the API (for UI).  This attribute is deprecated and will be removed in future versions.
	CurrentUserRoleID int32 `json:"current_user_role_id,omitempty"`
	// The list of role ID of the current user who triggered the API (for UI)
	CurrentUserRoleIds []int32 `json:"current_user_role_ids,omitempty"`
	// The number of the repositories under this project.
	RepoCount int32 `json:"repo_count,omitempty"`
	// The total number of charts under this project.
	ChartCount int32 `json:"chart_count,omitempty"`
	// The metadata of the project.
	Metadata *ProjectMetadata `json:"metadata,omitempty"`
	// The CVE allowlist of this project.
	// CveAllowlist *CveAllowlist `json:"cve_allowlist,omitempty"`
}

// ProjectMetadata the struct of harbor ProjectMetadata
type ProjectMetadata struct {
	// The public status of the project. The valid values are \"true\", \"false\".
	Public string `json:"public,omitempty"`
	// Whether content trust is enabled or not. If it is enabled, user can't pull unsigned images from this project. The valid values are \"true\", \"false\".
	EnableContentTrust string `json:"enable_content_trust,omitempty"`
	// Whether prevent the vulnerable images from running. The valid values are \"true\", \"false\".
	PreventVul string `json:"prevent_vul,omitempty"`
	// If the vulnerability is high than severity defined here, the images can't be pulled. The valid values are \"none\", \"low\", \"medium\", \"high\", \"critical\".
	Severity string `json:"severity,omitempty"`
	// Whether scan images automatically when pushing. The valid values are \"true\", \"false\".
	AutoScan string `json:"auto_scan,omitempty"`
	// Whether this project reuse the system level CVE allowlist as the allowlist of its own.  The valid values are \"true\", \"false\". If it is set to \"true\" the actual allowlist associate with this project, if any, will be ignored.
	ReuseSysCveAllowlist string `json:"reuse_sys_cve_allowlist,omitempty"`
	// The ID of the tag retention policy for the project
	RetentionID string `json:"retention_id,omitempty"`
}

// ListRepositories list repositories from a harbor with v2 api
func (hc *HarborClient2) ListRepositories(projectName string) ([]*Repository, error) {
	url := fmt.Sprintf("%s/projects/%s/repositories", hc.BasePath, projectName)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(hc.Auth.Username, hc.Auth.Password)
	klog.Infof("list repositories url = %s", url)
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
	// The description of the repository
	Description string `json:"description,omitempty"`
	// The count of the artifacts inside the repository
	ArtifactCount int64 `json:"artifact_count,omitempty"`
	// The count that the artifact inside the repository pulled
	PullCount int64 `json:"pull_count,omitempty"`
	// The creation time of the repository
	CreationTime time.Time `json:"creation_time,omitempty"`
	// The update time of the repository
	UpdateTime time.Time `json:"update_time,omitempty"`
}

// ListArtifacts list artifacts from a harbor with v2 api
func (hc *HarborClient2) ListArtifacts(projectName, repoName string) ([]*Artifact, error) {
	url := fmt.Sprintf("%s/projects/%s/repositories/%s/artifacts", hc.BasePath, projectName, repoName)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(hc.Auth.Username, hc.Auth.Password)
	klog.Infof("list artifacts url = %s", url)
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

// Artifact the struct of harbor Artifact
type Artifact struct {
	// The ID of the artifact
	ID int64 `json:"id,omitempty"`
	// The type of the artifact, e.g. image, chart, etc
	Type_ string `json:"type,omitempty"`
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
	// The digest of the icon
	Icon string `json:"icon,omitempty"`
	// The push time of the artifact
	PushTime time.Time `json:"push_time,omitempty"`
	// The latest pull time of the artifact
	PullTime    time.Time              `json:"pull_time,omitempty"`
	ExtraAttrs  map[string]interface{} `json:"extra_attrs,omitempty"`
	Annotations map[string]string      `json:"annotations,omitempty"`
	// References  []Reference            `json:"references,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
	// AdditionLinks *AdditionLinks         `json:"addition_links,omitempty"`
	// Labels        []Label                `json:"labels,omitempty"`
	// The overview of the scan result.
	// ScanOverview *ScanOverview `json:"scan_overview,omitempty"`
}

// Tag the struct of harbor Tag
type Tag struct {
	// The ID of the tag
	ID int64 `json:"id,omitempty"`
	// The ID of the repository that the tag belongs to
	RepositoryID int64 `json:"repository_id,omitempty"`
	// The ID of the artifact that the tag attached to
	ArtifactID int64 `json:"artifact_id,omitempty"`
	// The name of the tag
	Name string `json:"name,omitempty"`
	// The push time of the tag
	PushTime time.Time `json:"push_time,omitempty"`
	// The latest pull time of the tag
	PullTime time.Time `json:"pull_time,omitempty"`
	// The immutable status of the tag
	Immutable bool `json:"immutable,omitempty"`
	// The attribute indicates whether the tag is signed or not
	Signed bool `json:"signed,omitempty"`
}
