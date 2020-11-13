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

// HarborWhiteListExample ...
var HarborWhiteListExample = &WhiteList{
	Harbor: map[string][]string{
		"proj": {
			"test-mychart",
			"mychart",
		},
	},
	Registry: map[string][]string{
		"proj/chart-abc": {
			"v1",
			"v2",
		},
		"^test/chart-xyz$": {
			"v1",
		},
	},
}

// LoadWhiteList try to load whiteList form WhiteListFilePath
func LoadWhiteList() error {
	klog.Infof("Try to load whiteList form %s", WhiteListFilePath)
	body, err := ioutil.ReadFile(WhiteListFilePath)
	if err != nil {
		klog.Warningf("Read file : %s failed, the reason is : %s", WhiteListFilePath, err.Error())
	} else {
		if err := json.Unmarshal(body, GlobalWhiteList); err != nil {
			klog.Warningf("Unmarshal whitelist json failed, the reason is : %s", err.Error())
			return err
		}

		klog.Infof("Load whiteList content is %+v", GlobalWhiteList)
	}

	return nil
}
