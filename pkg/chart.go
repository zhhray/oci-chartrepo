package pkg

import (
	"context"
	"encoding/json"
	"errors"
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

	// 	data, err := GetChartData(name)
	data, err := PullOCIArtifact(name)
	if err != nil {
		return err
	}
	c.Response().Header().Set("Content-Type", "application/x-tar")
	_, err = c.Response().Write(data)
	return err

}

// GetChartData get the charts data from a chart name
// func GetChartData(name string) ([]byte, error) {
// 	ref := pathToRefCache[name]
// 	result, err := GlobalBackend.Hub.DownloadBlob(ref.Name, ref.Digest)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return ioutil.ReadAll(result)
// }

// PullOCIArtifact pull an artifact from harbor
// The code borrows from Helm's func PullChart
// https://github.com/helm/helm/blob/master/cmd/helm/chart_pull.go
// Microsoft has donated ORAS as a means to enable various client libraries with a way
// to push OCI Artifacts to OCI Conformant registries, see https://github.com/deislabs/oras
func PullOCIArtifact(name string) ([]byte, error) {
	ref := pathToRefCache[name]
	store := content.NewMemoryStore()
	artifact := fmt.Sprintf("%s/%s", GlobalBackend.Host, ref.Name)

	klog.Infof("============artifact====== %s", artifact)

	desc, _, err := oras.Pull(context.Background(), GlobalBackend.Resolver, artifact, store,
		oras.WithAllowedMediaTypes(KnownMediaTypes()),
		oras.WithPullEmptyNameAllowed(),
		oras.WithContentProvideIngester(store))
	if err != nil {
		klog.Error("=========err oras.Pull======", err)
		return nil, err
	}
	klog.Infof("=====desc====== %v", desc)

	manifestBytes, err := fetchBlob(store, &desc)
	if err != nil {
		klog.Error("=========err fetchBlob 111======", err)
		return nil, err
	}

	var manifest ocispec.Manifest
	err = json.Unmarshal(manifestBytes, &manifest)
	if err != nil {
		return nil, err
	}
	numLayers := len(manifest.Layers)
	if numLayers != 1 {
		return nil, errors.New(
			fmt.Sprintf("manifest does not contain exactly 1 layer (total: %d)", numLayers))
	}

	var contentLayer *ocispec.Descriptor
	for _, layer := range manifest.Layers {
		switch layer.MediaType {
		case HelmChartContentLayerMediaType:
			contentLayer = &layer
		}
	}
	if contentLayer == nil {
		return nil, errors.New(
			fmt.Sprintf("manifest does not contain a layer with mediatype %s", HelmChartContentLayerMediaType))
	}
	if contentLayer.Size == 0 {
		return nil, errors.New(
			fmt.Sprintf("manifest layer with mediatype %s is of size 0", HelmChartContentLayerMediaType))
	}

	out, err := fetchBlob(store, contentLayer)
	if err != nil {
		klog.Error("=========err fetchBlob 222 ======", err)
		return nil, err
	}

	return out, nil
}

// fetchBlob retrieves a blob from filesystem
func fetchBlob(store *content.Memorystore, desc *ocispec.Descriptor) ([]byte, error) {
	// reader, err := store.ReaderAt(GlobalBackend.Harbor.AuthContext, *desc)
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
