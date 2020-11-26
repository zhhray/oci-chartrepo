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

// LoadWhiteList try to load whiteList form WhiteListFilePath
func LoadWhiteList() error {
	klog.Infof("Try to load whiteList config file form %s", WhiteListFilePath)
	body, err := ioutil.ReadFile(WhiteListFilePath)
	if err != nil {
		klog.Warningf("Read file : %s failed, the reason is : %s", WhiteListFilePath, err.Error())
	} else {
		if err := json.Unmarshal(body, GlobalWhiteList); err != nil {
			klog.Warningf("Unmarshal whitelist json failed, the reason is : %s", err.Error())
			return err
		}

		klog.Infof("The whiteList config file was successfully loaded. Content is %+v", GlobalWhiteList)
	}

	return nil
}
