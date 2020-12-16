package pkg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"k8s.io/klog"
)

func genPath(name, version string) string {
	return fmt.Sprintf("%s-%s.tgz", name, version)
}

// LoadWhiteList try to load whiteList to GlobalWhiteList
func LoadWhiteList() error {
	if err := loadWhiteListFromConfigFile(); err != nil {
		return loadWhiteListFromProductBase()
	}

	if GlobalWhiteList.Mode != StrictMode && GlobalWhiteList.Mode != MatchMode {
		return loadWhiteListFromProductBase()
	}

	return nil
}

func loadWhiteListFromConfigFile() error {
	klog.Infof("Try to load whiteList config file form %s", WhiteListFilePath)
	body, err := ioutil.ReadFile(WhiteListFilePath)
	if err != nil {
		klog.Warningf("Read file : %s failed, the reason is : %s", WhiteListFilePath, err.Error())
		return err
	}

	if err := json.Unmarshal(body, GlobalWhiteList); err != nil {
		klog.Warningf("Unmarshal whitelist json failed, the reason is : %s", err.Error())
		return err
	}

	klog.Infof("The whiteList config file was successfully loaded. Content is %+v", GlobalWhiteList)
	return nil
}

func loadWhiteListFromProductBase() error {
	klog.Infof("Try to get chartVersions from ProductBase.")
	pb, err := getProductBase("base")
	if err != nil {
		klog.Warningf("Get productbase failed, the reason is : %s", err.Error())
		return err
	}

	if pb != nil && pb.Spec != nil && len(pb.Spec.ChartVersions) > 0 {
		GlobalWhiteList.Mode = StrictMode
		GlobalWhiteList.ChartVersions = pb.Spec.ChartVersions

		klog.Infof("Success get chartVersions from ProductBase and set it to whitelist. Content is %+v", GlobalWhiteList.ChartVersions)
	}

	return nil
}

func loadObjectsFromCache() []HelmOCIConfig {
	objects := make([]HelmOCIConfig, 0)

	for _, obj := range refToChartCache {
		objects = append(objects, *obj)
	}

	return objects
}

// objectAlreadyExist determine atf duplication in []HelmOCIConfig
func objectAlreadyExist(objects []HelmOCIConfig, digest string) bool {
	if digest == "" {
		return false
	}

	for _, obj := range objects {
		if obj.Digest == digest {
			return true
		}
	}

	return false
}
