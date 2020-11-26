package pkg

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/heroku/docker-registry-client/registry"
	"k8s.io/klog"
)

// RegistryHub is registry backend will implement Hub interface
type RegistryHub struct {
	Registry *registry.Registry
}

// ListObjects parser all helm chart basic info from oci manifest
// skip all manifests that are not helm type
func (r *RegistryHub) ListObjects() ([]HelmOCIConfig, error) {
	objects := make([]HelmOCIConfig, 0)
	var isBreak bool
	var err error
	if len(GlobalWhiteList.Registry) > 0 {
		for image, tags := range GlobalWhiteList.Registry {
			if len(tags) > 0 {
				for _, tag := range tags {
					objects, isBreak, err = r.listObjectsByImageAndTag(image, tag, objects)
					if err != nil {
						if isBreak {
							klog.Error("err lsit objects by image and tag", err)
							break
						}

						klog.Warning("err lsit objects by image and tag", err)
					}
				}
			} else {
				objects, isBreak, err = r.listObjectsByImage(image, objects)
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
		repositories, err := r.Registry.Repositories()
		if err != nil {
			return nil, err
		}

		for _, image := range repositories {
			objects, isBreak, err = r.listObjectsByImage(image, objects)
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

// listObjectsByImage return []HelmOCIConfig, isBreak, error
func (r *RegistryHub) listObjectsByImage(image string, objects []HelmOCIConfig) ([]HelmOCIConfig, bool, error) {
	tags, err := r.Registry.Tags(image)
	if err != nil {
		klog.Error("err list tags for repo: ", err)
		// You can list Repositories, but the API returns UNAUTHORIZED or PROJECT_POLICY_VIOLATION when you list tags for a repository
		if strings.Contains(err.Error(), "repository name not known to registry") ||
			strings.Contains(err.Error(), "UNAUTHORIZED") ||
			strings.Contains(err.Error(), "PROJECT_POLICY_VIOLATION") {

			// need break
			return objects, true, err
		}

		return objects, false, err
	}

	for _, tag := range tags {
		var isBreak bool
		var err error
		objects, isBreak, err = r.listObjectsByImageAndTag(image, tag, objects)
		if err != nil {
			if isBreak {
				klog.Error("err lsit objects by image and tag", err)
				// need break
				break
			}

			klog.Warning("err lsit objects by image and tag", err)
		}
	}

	return objects, false, nil
}

// listObjectsByImageAndTag return []HelmOCIConfig, isBreak, error
func (r *RegistryHub) listObjectsByImageAndTag(image, tag string, objects []HelmOCIConfig) ([]HelmOCIConfig, bool, error) {
	manifest, err := r.Registry.OCIManifestV1(image, tag)
	if err != nil {
		klog.Warning("err get manifest for tag: ", err)
		// You can list tags, but the API returns UNAUTHORIZED or PROJECT_POLICY_VIOLATION when you get manifest for a tag
		if strings.Contains(err.Error(), "UNAUTHORIZED") ||
			strings.Contains(err.Error(), "PROJECT_POLICY_VIOLATION") {
			// need break
			return objects, true, err
		}

		// continue here.
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
	result, err := r.Registry.DownloadBlob(image, digest)
	if err != nil {
		return objects, false, err
	}
	body, err := ioutil.ReadAll(result)
	if err != nil {
		return objects, false, err
	}
	result.Close()

	cfg := &HelmOCIConfig{}
	err = json.Unmarshal(body, cfg)
	if err != nil {
		return objects, false, err
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
