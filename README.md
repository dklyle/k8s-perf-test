# perf
performance testing starting and stopping pods and containers written in go

## build
`make all` with build all the executables and put them in `bin`

These are the executables for testing:
* `docker` runs using the Docker CLI for `docker run` and `docker stop`
* `k8s` runs using the Kubernetes CLI, `kubectl`, for `kubectl run pod` and
`kubectl delete pod`
* `k8s-api` runs using the Kubernetes API for starting and stopping pods

These executables should be copied to the leader node of a Kubernetes cluster.


## cluster configuration
A couple of customizations are required to a standard Kubernetes cluster.
1. For all executables other than `docker-test`

`kubectl proxy --port=8090 &`

2. For `k8s-api-test` and `k8s-api-preempt-test` configuration of the metrics
API is required. The simplest was to do this is install and configure Prometheus:

```
git clone https://github.com/coreos/kube-prometheus
kubectl create -f kube-prometheus/manifests/
```


## running
The script `run-trials.sh` will run a series of tests with increasing numbers of
containers or pods. There are three parameters needed.

`./run-trials.sh PROGRAM GRACE CSV`
Where:
* PROGRAM (with PATH, ./program_name for current directory), this is one of the
executables made in [build](#build)
* GRACE is the number of seconds for termination grace period
* CSV is either `true` or `false` indicating to write results to file in csv
format or not

This is executed on the leader node of the cluster. Using something like:

`./run-trials.sh ./k8s-api-test 0 true`

For simplicity of tracking the results, it is suggested that you make a different
directory for each trial type that you plan to run, e.g., `k8s-api-trials` and
`docker-trials`. In those directories place the respective executable from the
[build](#build) section.


## data
If `CSV` was specified as `true` when executing `run-trials.sh`, then
`run-trials.sh` will output a set of directories, one for each number of
container or pods that was started simultaneously. These directories will be
named by the nanosecond unix timestamp when they began.
In each directory there will be one csv file.

For ease, make a directory with the corresponding `GRACE` value entitled
g`GRACE`.

Included in the source code for this project is the `results` directory where
some data handling scripts are included, most notably `concat-data.sh`.

`./concat-data.sh TEST_TYPE GRACE1 [GRACE2]`
* TEST_TYPE is one of k8s-api, k8s, docker
* GRACE1 the grace value in the file names
* GRACE2 the grace value in the file names this value is optional if stitching
together two grace values into one result.
