# redis-cluster-operator
Redis Cluster Operator manages [Redis Cluster](https://redis.io/topics/cluster-spec) atop Kubernetes.

The operator itself is built with the [Operator framework](https://github.com/operator-framework/operator-sdk).

![Redis Cluster atop Kubernetes](/static/redis-cluster.png)

Each master node and its slave nodes is managed by a statefulSet, create a headless svc for each statefulSet,
and create a clusterIP service for all nodes.

Each statefulset uses PodAntiAffinity to ensure that the master and slaves are dispersed on different nodes.
At the same time, when the operator selects the master in each statefulset, it preferentially select the pod
with different k8s nodes as master.


## Table of Contents

- [Overview](#redis-cluster-operator)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [To Deploy on the Cluster](#to-deploy-on-the-cluster)
  - [To Uninstall](#to-uninstall)
- [Project Distribution](#project-distribution)
  - [Build Installer](#build-the-installer)
  - [Using the Installer](#using-the-installer)
- [Usage](#usage)
  - [Deploy a Sample Redis Cluster](#deploy-a-sample-redis-cluster)
  - [Scaling Up the Redis Cluster](#scaling-up-the-redis-cluster)
  - [Scaling Down the Redis Cluster](#scaling-down-the-redis-cluster)
- [Contributing](#contributing)

## Getting Started

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=tyk/redis-cluster-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/redis-cluster-operator:tag
```

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=tyk/redis-cluster-operator:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/redis-cluster-operator/<tag or branch>/dist/install.yaml
```

### Usage
#### Deploy a sample Redis Cluster

NOTE: **Only the redis cluster that use persistent storage(pvc) can recover after accidental deletion or rolling update.Even if you do not use persistence(like rdb or aof), you need to set pvc for redis.**

```
$ kubectl apply -f deploy/example/redis.kun_v1alpha1_distributedrediscluster_cr.yaml
```

Verify that the cluster instances and its components are running.
```
$ kubectl get distributedrediscluster
NAME                              MASTERSIZE   STATUS    AGE
example-distributedrediscluster   3            Scaling   11s

$ kubectl get all -l redis.kun/name=example-distributedrediscluster
NAME                                          READY   STATUS    RESTARTS   AGE
pod/drc-example-distributedrediscluster-0-0   1/1     Running   0          2m48s
pod/drc-example-distributedrediscluster-0-1   1/1     Running   0          2m8s
pod/drc-example-distributedrediscluster-1-0   1/1     Running   0          2m48s
pod/drc-example-distributedrediscluster-1-1   1/1     Running   0          2m13s
pod/drc-example-distributedrediscluster-2-0   1/1     Running   0          2m48s
pod/drc-example-distributedrediscluster-2-1   1/1     Running   0          2m15s

NAME                                        TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)              AGE
service/example-distributedrediscluster     ClusterIP   172.17.132.71   <none>        6379/TCP,16379/TCP   2m48s
service/example-distributedrediscluster-0   ClusterIP   None            <none>        6379/TCP,16379/TCP   2m48s
service/example-distributedrediscluster-1   ClusterIP   None            <none>        6379/TCP,16379/TCP   2m48s
service/example-distributedrediscluster-2   ClusterIP   None            <none>        6379/TCP,16379/TCP   2m48s

NAME                                                     READY   AGE
statefulset.apps/drc-example-distributedrediscluster-0   2/2     2m48s
statefulset.apps/drc-example-distributedrediscluster-1   2/2     2m48s
statefulset.apps/drc-example-distributedrediscluster-2   2/2     2m48s

$ kubectl get distributedrediscluster
NAME                              MASTERSIZE   STATUS    AGE
example-distributedrediscluster   3            Healthy   4m
```

#### Scaling Up the Redis Cluster

Increase the masterSize to trigger the scaling up.

```
apiVersion: redis.kun/v1alpha1
kind: DistributedRedisCluster
metadata:
  annotations:
    # if your operator run as cluster-scoped, add this annotations
    redis.kun/scope: cluster-scoped
  name: example-distributedrediscluster
spec:
  # Increase the masterSize to trigger the scaling.
  masterSize: 4
  ClusterReplicas: 1
  image: redis:5.0.4-alpine
```

#### Scaling Down the Redis Cluster

Decrease the masterSize to trigger the scaling down.

```
apiVersion: redis.kun/v1alpha1
kind: DistributedRedisCluster
metadata:
  annotations:
    # if your operator run as cluster-scoped, add this annotations
    redis.kun/scope: cluster-scoped
  name: example-distributedrediscluster
spec:
  # Decrease the masterSize to trigger the scaling.
  masterSize: 3
  ClusterReplicas: 1
  image: redis:5.0.4-alpine
```

## Contributing

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)
