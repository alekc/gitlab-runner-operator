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
```shell
make docker-build
kind load docker-image controller:dev
make deploy IMG=controller:dev
```




