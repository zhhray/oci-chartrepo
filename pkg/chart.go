package pkg

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"
	"github.com/labstack/echo/v4"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"k8s.io/klog"
)

// GetChartHandler get the charts data api
func GetChartHandler(c echo.Context) error {
	name := c.Param("name")
	data, err := pullOCIArtifact(name)
	if err != nil {
		klog.Error("pull OCI artifact err, ", err)
		return err
	}
	c.Response().Header().Set("Content-Type", "application/x-tar")
	_, err = c.Response().Write(data)
	return err

}

// pullOCIArtifact pull an artifact from harbor
// The code borrows from Helm's func PullChart
// https://github.com/helm/helm/blob/master/cmd/helm/chart_pull.go
// Microsoft has donated ORAS as a means to enable various client libraries with a way
// to push and pull OCI Artifacts to and from OCI Conformant registries, see https://github.com/deislabs/oras
func pullOCIArtifact(name string) ([]byte, error) {
	ref, exist := pathToRefCache[name]
	if !exist {
		return nil, fmt.Errorf("can not get ref from cache by name : %s", name)
	}

	store := content.NewMemoryStore()
	artifact := fmt.Sprintf("%s/%s", GlobalBackend.Host, ref.Name)

	klog.Infof("start pulling artifact %s", artifact)
	desc, _, err := oras.Pull(context.Background(), GlobalBackend.Resolver, artifact, store,
		oras.WithAllowedMediaTypes(KnownMediaTypes()),
		oras.WithPullEmptyNameAllowed(),
		oras.WithContentProvideIngester(store))
	if err != nil {
		return nil, err
	}

	manifestBytes, err := fetchBlob(store, &desc)
	if err != nil {
		return nil, err
	}

	var manifest ocispec.Manifest
	err = json.Unmarshal(manifestBytes, &manifest)
	if err != nil {
		return nil, err
	}
	numLayers := len(manifest.Layers)
	if numLayers != 1 {
		return nil, fmt.Errorf("manifest does not contain exactly 1 layer (total: %d)", numLayers)
	}

	var contentLayer *ocispec.Descriptor
	for _, layer := range manifest.Layers {
		switch layer.MediaType {
		case HelmChartContentLayerMediaType:
			contentLayer = &layer
		}
	}
	if contentLayer == nil {
		return nil, fmt.Errorf("manifest does not contain a layer with mediatype %s", HelmChartContentLayerMediaType)
	}
	if contentLayer.Size == 0 {
		return nil, fmt.Errorf("manifest layer with mediatype %s is of size 0", HelmChartContentLayerMediaType)
	}

	out, err := fetchBlob(store, contentLayer)
	if err != nil {
		return nil, err
	}
	klog.Infof("pull artifact %s complate", artifact)

	return out, nil
}

// fetchBlob retrieves a blob from filesystem
func fetchBlob(store *content.Memorystore, desc *ocispec.Descriptor) ([]byte, error) {
	reader, err := store.ReaderAt(context.Background(), *desc)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	bytes := make([]byte, desc.Size)
	_, err = reader.ReadAt(bytes, 0)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}
