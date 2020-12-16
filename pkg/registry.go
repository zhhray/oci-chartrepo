package pkg

import (
	"encoding/json"
	"io/ioutil"
	"time"

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
	st := time.Now()
	objects := loadObjectsFromCache()
	var isBreak bool
	var err error
	if GlobalWhiteList.Mode == StrictMode {
		for _, cv := range GlobalWhiteList.ChartVersions {
			// cv.Name: acp/chart-alauda-container-platform
			// if strict: true, will ignore err and not break loop
			objects, _, err = r.listObjectsByImageAndTag(cv.Name, cv.Version, objects)
			if err != nil {
				klog.Warning("err lsit objects by image and tag", err)
			}
		}
	} else {
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
					objects, _ = r.listObjectsByImage(image, objects)
				}
			}
		} else {
			repositories, err := r.Registry.Repositories()
			if err != nil {
				return nil, err
			}

			for _, image := range repositories {
				objects, _ = r.listObjectsByImage(image, objects)
			}
		}
	}

	klog.Infof("======repo length is %d", len(repoMap))
	klog.Infof("======objects length is %d", len(objects))
	ed := time.Now()
	sub := ed.Sub(st)
	klog.Infof("======listObjects use %f min", sub.Minutes())
	return objects, nil
}

// listObjectsByImage return []HelmOCIConfig, error
func (r *RegistryHub) listObjectsByImage(image string, objects []HelmOCIConfig) ([]HelmOCIConfig, error) {
	tags, err := r.Registry.Tags(image)
	if err != nil {
		klog.Error("err list tags for repo: ", err)
		return objects, err
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

	return objects, nil
}

// listObjectsByImageAndTag return []HelmOCIConfig, isBreak, error
func (r *RegistryHub) listObjectsByImageAndTag(image, tag string, objects []HelmOCIConfig) ([]HelmOCIConfig, bool, error) {
	repoMap[image+":"+tag] = image + ":" + tag

	manifest, err := r.Registry.OCIManifestV1(image, tag)
	if err != nil {
		klog.Warning("err get manifest for tag: ", err)
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

	if objectAlreadyExist(objects, manifest.Layers[0].Digest.Encoded()) {
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

	ref := image + ":" + tag
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
