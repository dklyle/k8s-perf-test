# perf
performance testing starting and stopping pods and containers written in go

## build
`make all` with build all the executables and put them in `bin`

These are the executables for testing:
* `docker` runs using the Docker CLI for `docker run` and `docker stop`
* `k8s` runs using the Kubernetes CLI, `kubectl`, for `kubectl run pod` and `kubectl delete pod`
* `k8s-api` runs using the Kubernetes API for starting and stopping pods

These executables should be copied to the Master node of a Kubernetes cluster.

## running
The script `run-trials.sh` will run a series of tests with increasing numbers of containers or pods. There are three parameters needed.

`./run-trials.sh PROGRAM GRACE CSV`
Where:
* PROGRAM (with PATH, ./program_name for current directory), this is one of the executables made in [build](#build)
* GRACE is the number of seconds for termination grace period
* CSV is either `true` or `false` indicating to write results to file in csv format or not

## data
If `CSV` was specified as `true` when executing `run-trials.sh`, then
`run-trials.sh` will output a set of directories, one for each number of container or pods that was started simultaneously. These directories will be named by the nanosecond unix timestamp when they began.
In each directory there will be one csv file.
...
