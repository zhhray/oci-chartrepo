module github.com/alauda/oci-chartrepo

go 1.13

require (
	github.com/containerd/containerd v1.3.4
	github.com/deislabs/oras v0.8.1
	github.com/ghodss/yaml v1.0.0
	github.com/heroku/docker-registry-client v0.0.0-20190909225348-afc9e1acc3d5
	github.com/labstack/echo/v4 v4.1.17
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	helm.sh/helm/v3 v3.3.1
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v0.18.8
	k8s.io/klog v1.0.0
)

replace github.com/heroku/docker-registry-client => github.com/alauda/docker-registry-client v0.0.0-20200917062349-081af988aae6
