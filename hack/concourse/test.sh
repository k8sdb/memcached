#!/bin/bash

set -x -e

# start docker and log-in to docker-hub
entrypoint.sh
docker login --username=$DOCKER_USER --password=$DOCKER_PASS
docker run hello-world

# install python pip
apt-get update >/dev/null
apt-get install -y python python-pip >/dev/null

# install kubectl
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl &>/dev/null
chmod +x ./kubectl
mv ./kubectl /bin/kubectl

# install onessl
curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.3.0/onessl-linux-amd64 &&
  chmod +x onessl &&
  mv onessl /usr/local/bin/

# install pharmer
pushd /tmp
curl -LO https://cdn.appscode.com/binaries/pharmer/0.1.0-rc.3/pharmer-linux-amd64
chmod +x pharmer-linux-amd64
mv pharmer-linux-amd64 /bin/pharmer
popd

function cleanup() {
  # delete cluster on exit
  pharmer get cluster || true
  pharmer delete cluster $NAME || true
  pharmer get cluster || true
  sleep 120 || true
  pharmer apply $NAME || true
  pharmer get cluster || true

  # delete docker image on exit
  curl -LO https://raw.githubusercontent.com/appscodelabs/libbuild/master/docker.py || true
  chmod +x docker.py || true
  ./docker.py del_tag kubedbci mc-operator $CUSTOM_OPERATOR_TAG
}
trap cleanup EXIT

# copy memcached to $GOPATH
mkdir -p $GOPATH/src/github.com/kubedb
cp -r memcached $GOPATH/src/github.com/kubedb

pushd $GOPATH/src/github.com/kubedb/memcached
# name of the cluster
# nameing is based on repo+commit_hash
NAME=memcached-$(git rev-parse --short HEAD)

./hack/builddeps.sh
export APPSCODE_ENV=dev
export DOCKER_REGISTRY=kubedbci
./hack/docker/mc-operator/make.sh build
./hack/docker/mc-operator/make.sh push
popd

# create credential file for pharmer
cat >cred.json <<EOF
{
        "token" : "$TOKEN"
}
EOF

# create cluster using pharmer
# note: make sure the zone supports volumes, not all regions support that
# "We're sorry! Volumes are not available for Droplets on legacy hardware in the NYC3 region"
pharmer create credential --from-file=cred.json --provider=DigitalOcean cred
pharmer create cluster $NAME --provider=digitalocean --zone=nyc1 --nodes=2gb=1 --credential-uid=cred --kubernetes-version=v1.10.0
pharmer apply $NAME
pharmer use cluster $NAME
#wait for cluster to be ready
sleep 120
kubectl get nodes

pushd $GOPATH/src/github.com/kubedb/memcached

# run tests
source ./hack/deploy/setup.sh --docker-registry=kubedbci
./hack/make.py test e2e --v=1 --selfhosted-operator=true
