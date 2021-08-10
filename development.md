## Kind cluster

### Spinning 
To spin up a local testing kind cluster (1 master node, 2 workers), you can run 

`kind create cluster --config hack/kind-config.yaml --image=kindest/node:v1.22.0`

You can use --image flag to specify the cluster version you want, e.g. --image=kindest/node:v1.17.2, the supported version are listed [here](https://hub.docker.com/r/kindest/node/tags)

### Install cert manager
In order to test our webhook implementation, sadly we need to have a support of cert-manager

```
kubectl config set-context kind-gitlab-runner-cluster
kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
```

### Crd
Install crd to the cluster
`make install`

### Docker image 
Note: you need to specify an image name during the build. If the image has `latest` tag, then kind will attempt to download
it even if you have loaded it through `kind load docker-image`

```shell
make docker-build controller:dev
kind load docker-image controller:dev
make deploy IMG=controller:dev
```




