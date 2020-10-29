module github.com/alauda/oci-chartrepo

go 1.13

require (
	github.com/antihax/optional v1.0.0
	github.com/containerd/containerd v1.3.4
	github.com/deislabs/oras v0.8.1
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-openapi/errors v0.19.2
	github.com/go-openapi/runtime v0.19.4
	github.com/go-openapi/strfmt v0.19.3
	github.com/go-openapi/swag v0.19.5
	github.com/heroku/docker-registry-client v0.0.0-20190909225348-afc9e1acc3d5
	github.com/labstack/echo/v4 v4.1.17
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	helm.sh/helm/v3 v3.3.1
	k8s.io/klog v1.0.0
)

replace github.com/heroku/docker-registry-client => github.com/alauda/docker-registry-client v0.0.0-20200917062349-081af988aae6
