#!/bin/bash
set -eou pipefail
IFS=$'\n\t'

DEMODATA_DIR=$(pwd)/demodata
rm -rf "${DEMODATA_DIR}" && mkdir -p "${DEMODATA_DIR}"

docker network create kind || true

REGISTRY_IP=$(docker network inspect kind |
  gojq -r '.[].IPAM.Config[].Subnet |
                          capture("^(?<octet1and2>(?:\\d{1,3}\\.){2})(?:\\d{1,3})\\.(?:\\d{1,3})/(?:\\d{1,3})$") |
                          .octet1and2 + "0.10"')

REGISTRY_PORT=5000
# Use a domain so it can be access on a Mac
REGISTRY_ADDRESS=registry
REGISTRY_CERTS_DIR="${DEMODATA_DIR}/certs"
REGISTRY_AUTH_DIR="${DEMODATA_DIR}/auth"
REGISTRY_USERNAME=testuser
REGISTRY_PASSWORD=testpassword

# Create certs for the registry
rm -rf "${REGISTRY_CERTS_DIR}"
mkdir -p "${REGISTRY_CERTS_DIR}"

OPENSSL_BIN="${OPENSSL_BIN:=openssl}"

${OPENSSL_BIN} req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
  -keyout "${REGISTRY_CERTS_DIR}/ca.key" -out "${REGISTRY_CERTS_DIR}/ca.crt" -subj "/CN=${REGISTRY_ADDRESS}" \
  -addext "subjectAltName=DNS:${REGISTRY_ADDRESS},IP:${REGISTRY_IP}"

sudo mkdir -p "/etc/docker/certs.d/${REGISTRY_IP}:${REGISTRY_PORT}"
sudo cp -f "${REGISTRY_CERTS_DIR}/ca.crt" "/etc/docker/certs.d/${REGISTRY_IP}:${REGISTRY_PORT}"

rm -rf "${REGISTRY_AUTH_DIR}"
mkdir -p "${REGISTRY_AUTH_DIR}"
docker container run --rm \
  --entrypoint htpasswd \
  httpd:2 -Bbn "${REGISTRY_USERNAME}" "${REGISTRY_PASSWORD}" >"${REGISTRY_AUTH_DIR}/htpasswd"

docker container rm -fv registry || true
docker container run --rm -d \
  --name registry \
  --ip "${REGISTRY_IP}" \
  --network kind \
  -v "${REGISTRY_CERTS_DIR}":/certs \
  -e REGISTRY_HTTP_ADDR="${REGISTRY_IP}:${REGISTRY_PORT}" \
  -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/ca.crt \
  -e REGISTRY_HTTP_TLS_KEY=/certs/ca.key \
  -v "${REGISTRY_AUTH_DIR}":/auth \
  -e "REGISTRY_AUTH=htpasswd" \
  -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" \
  -e REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd \
  -p "${REGISTRY_PORT}":"${REGISTRY_PORT}" \
  registry:2

cat <<EOF >"${DEMODATA_DIR}/containerd-config.toml"
# explicitly use v2 config format
version = 2

[proxy_plugins]
# fuse-overlayfs is used for rootless
[proxy_plugins."fuse-overlayfs"]
  type = "snapshot"
  address = "/run/containerd-fuse-overlayfs.sock"

[plugins."io.containerd.grpc.v1.cri".containerd]
  # save disk space when using a single snapshotter
  discard_unpacked_layers = true
  # explicitly use default snapshotter so we can sed it in entrypoint
  snapshotter = "overlayfs"
  # explicit default here, as we're configuring it below
  default_runtime_name = "runc"
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
  # set default runtime handler to v2, which has a per-pod shim
  runtime_type = "io.containerd.runc.v2"
  # Generated by "ctr oci spec" and modified at base container to mount poduct_uuid
  base_runtime_spec = "/etc/containerd/cri-base.json"
  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
    # use systemd cgroup by default
    SystemdCgroup = true

# Setup a runtime with the magic name ("test-handler") used for Kubernetes
# runtime class tests ...
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.test-handler]
  # same settings as runc
  runtime_type = "io.containerd.runc.v2"
  base_runtime_spec = "/etc/containerd/cri-base.json"
  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.test-handler.options]
    SystemdCgroup = true

[plugins."io.containerd.grpc.v1.cri"]
  # use fixed sandbox image
  sandbox_image = "registry.k8s.io/pause:3.7"
  # allow hugepages controller to be missing
  # see https://github.com/containerd/cri/pull/1501
  tolerate_missing_hugepages_controller = true
  # restrict_oom_score_adj needs to be true when running inside UserNS (rootless)
  restrict_oom_score_adj = false

[plugins."io.containerd.grpc.v1.cri".registry]
  [plugins."io.containerd.grpc.v1.cri".registry.configs]
    [plugins."io.containerd.grpc.v1.cri".registry.configs."${REGISTRY_ADDRESS}:${REGISTRY_PORT}".tls]
      insecure_skip_verify = true
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
    [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
      endpoint = ["https://${REGISTRY_ADDRESS}:${REGISTRY_PORT}","https://registry-1.docker.io"]
    [plugins."io.containerd.grpc.v1.cri".registry.mirrors."*"]
      endpoint = ["https://${REGISTRY_ADDRESS}:${REGISTRY_PORT}"]
    # Enable registry.k8s.io as the primary mirror for k8s.gcr.io
    # See: https://github.com/kubernetes/k8s.io/issues/3411
    [plugins."io.containerd.grpc.v1.cri".registry.mirrors."k8s.gcr.io"]
      endpoint = ["https://registry.k8s.io", "https://k8s.gcr.io",]
EOF

cat <<EOF >"${DEMODATA_DIR}/image-credential-provider-config.yaml"
apiVersion: kubelet.config.k8s.io/v1beta1
kind: CredentialProviderConfig
providers:
- name: static-credential-provider
  matchImages:
  - "${REGISTRY_ADDRESS}:${REGISTRY_PORT}"
  - "*"
  - "*.*"
  - "*.*.*"
  - "*.*.*.*"
  - "*.*.*.*.*"
  - "*.*.*.*.*.*"
  defaultCacheDuration: "1m"
  apiVersion: credentialprovider.kubelet.k8s.io/v1beta1
EOF

mkdir -p "${DEMODATA_DIR}/image-credential-provider/"

cat <<EOF >"${DEMODATA_DIR}/image-credential-provider/static-credential-provider"
#!/usr/bin/env bash

echo "Got Request: " >> /etc/kubernetes/image-credential-provider/req.txt
echo "\$(</dev/stdin)" >> /etc/kubernetes/image-credential-provider/req.txt

# This is an initial provider that returns a dummy reponse and will be replaced after the cluster starts up
echo '{
  "kind":"CredentialProviderResponse",
  "apiVersion":"credentialprovider.kubelet.k8s.io/v1beta1",
  "cacheKeyType":"Registry",
  "cacheDuration":"0s",
  "auth":{
    "${REGISTRY_ADDRESS}:${REGISTRY_PORT}": {"username":"${REGISTRY_USERNAME}","password":"${REGISTRY_PASSWORD}"},
    "docker.io": {"username":"${REGISTRY_USERNAME}","password":"${REGISTRY_PASSWORD}"},
    "*.*": {"username":"","password":""}
  }
}'
EOF
chmod +x "${DEMODATA_DIR}/image-credential-provider/static-credential-provider"

cat <<EOF >"${DEMODATA_DIR}/kind-config.yaml"
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        image-credential-provider-config: /etc/kubernetes/image-credential-provider-config.yaml
        image-credential-provider-bin-dir: /etc/kubernetes/image-credential-provider/
        v: "6"
  extraMounts:
  - hostPath: ${DEMODATA_DIR}/containerd-config.toml
    containerPath: /etc/containerd/config.toml
  - hostPath: ${DEMODATA_DIR}/image-credential-provider-config.yaml
    containerPath: /etc/kubernetes/image-credential-provider-config.yaml
  # this directory and any configured providers need to exist during Kubelet's startup
  - hostPath: ${DEMODATA_DIR}/image-credential-provider/
    containerPath: /etc/kubernetes/image-credential-provider/
EOF

kind delete clusters image-credential-provider-test || true
kind create cluster --config="${DEMODATA_DIR}/kind-config.yaml" --name image-credential-provider-test

docker image pull nginx:latest
docker image tag docker.io/library/nginx:latest "${REGISTRY_IP}:${REGISTRY_PORT}/library/nginx:latest"

echo "${REGISTRY_PASSWORD}" | docker login -u "${REGISTRY_USERNAME}" --password-stdin "${REGISTRY_IP}:${REGISTRY_PORT}"
docker image push "${REGISTRY_IP}:${REGISTRY_PORT}/library/nginx:latest"

# Retag and push with a tag that doesn't exist in docker.io to test Containerd mirror config
docker image tag docker.io/library/nginx:latest "${REGISTRY_IP}:${REGISTRY_PORT}/library/nginx:$(whoami)"
docker image push "${REGISTRY_IP}:${REGISTRY_PORT}/library/nginx:$(whoami)"

# Wait for KIND to startup
sleep 10s

# Create a Pod with an image that is pulled from the registry
kubectl run nginx-latest --image=registry:5000/library/nginx:latest --image-pull-policy=Always
# Create a Pod with an image tag that does not exist in docker.io and exercise Containerd mirror
kubectl run nginx-mirror --image="docker.io/library/nginx:$(whoami)" --image-pull-policy=Always
# Create a Pod with an image tag that does not exist in registry:5000, but exists in docker.io
kubectl run nginx-stable --image=docker.io/library/nginx:stable --image-pull-policy=Always

kubectl wait --for=condition=Ready pods/nginx-latest pods/nginx-stable pods/nginx-mirror
