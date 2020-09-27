package pkg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func genPath(name, version string) string {
	return fmt.Sprintf("%s-%s.tgz", name, version)
}

func AnalyseResitryOptions(opts *RegistryOptions) error {
	// Scheme to use for connecting to the registry.
	if _, err := os.Stat(SecretCfgPath); !os.IsNotExist(err) {
		// Get user info from SecretCfgPath
		body, err := ioutil.ReadFile(SecretCfgPath)
		if err != nil {
			return err
		}
		var cfg DockerSecretCfg
		if err := json.Unmarshal(body, &cfg); err != nil {
			return err
		}

		for k, v := range cfg.Auths {
			if matchURL(opts.URL, k) {
				opts.Username = v.Username
				opts.Password = v.Password

				break
			}
		}
	}

	opts.Scheme = estimateScheme(opts.Scheme, opts.URL)

	return nil
}

func estimateScheme(scheme, url string) string {
	scheme = strings.ToLower(scheme)

	if scheme == SchemeTypeHTTP || scheme == SchemeTypeHTTPS {
		return scheme
	}

	if url == "" {
		return SchemeTypeHTTP
	}

	if strings.HasPrefix(url, PrefixHTTP) {
		return SchemeTypeHTTP
	} else if strings.HasPrefix(url, PrefixHTTPS) {
		return SchemeTypeHTTPS
	}

	return SchemeTypeHTTP
}

func matchURL(source, target string) bool {
	if strings.HasPrefix(source, PrefixHTTP) {
		source = source[len(PrefixHTTP):]
	} else if strings.HasPrefix(source, PrefixHTTPS) {
		source = source[len(PrefixHTTPS):]
	}

	if strings.HasPrefix(target, PrefixHTTP) {
		target = target[len(PrefixHTTP):]
	} else if strings.HasPrefix(target, PrefixHTTPS) {
		target = target[len(PrefixHTTPS):]
	}

	return source == target
}
