# oci-chartrepo
Helm 3 supports OCI for package distribution. Chart packages are able to be stored and shared across OCI-based registries. [Here](https://helm.sh/docs/topics/registries/)

However, many times you will want to access them through the CHART repository API (eg, ChartMuseum).

oci-chartrepo as an adapter that supports the transformation of OCI data structures into standard chart repository data structures.

## build
```sh
# supports amd64 and arm64
make linux-build
```

## run locally
```sh
docker build -t oci-chart-registry .
docker run -d --restart=always --name oci-chart-registry \
-p 8088:8080 \
oci-chart-registry --storage=registry --storage-registry-repo={your_registry_addr} --port=8080

# if your registry is HTTPS and user login is required, a file in dockerconfigjson(kubernetes secret type) format needs to be mounted to container /etc/secret/dockercfg
docker build -t oci-chart-registry .
docker run -d --restart=always --name oci-chart-registry \
-p 8088:8080 \
-v ~/dockercfg:/etc/secret/dockercfg \
oci-chart-registry --storage=registry --storage-registry-repo={your_registry_addr} --port=8080

# here ~/dockercfg can be generated by the following command：
kubectl create secret docker-registry myregistrykey --docker-server=DOCKER_REGISTRY_SERVER --docker-username=DOCKER_USER --docker-password=DOCKER_PASSWORD --docker-email=DOCKER_EMAIL
```