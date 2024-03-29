BIN_DIR = bin
SRC_DIR = src
DOCKER_TEST = ${BIN_DIR}/docker-test
K8S_TEST = ${BIN_DIR}/k8s-test
K8S_API_TEST = ${BIN_DIR}/k8s-api-test
K8S_API_PREEMPT_TEST = ${BIN_DIR}/k8s-api-preempt-test

$(shell mkdir -p ${BIN_DIR})

all: k8s-api-preempt k8s-api k8s docker
k8s-api-preempt:
	go build ${SRC_DIR}/k8s-api-preempt-test.go ${SRC_DIR}/csv.go ${SRC_DIR}/types.go
	mv k8s-api-preempt-test ${K8S_API_PREEMPT_TEST}
k8s-api:
	go build ${SRC_DIR}/k8s-api-test.go ${SRC_DIR}/csv.go ${SRC_DIR}/types.go
	mv k8s-api-test ${K8S_API_TEST}
k8s:
	go build ${SRC_DIR}/k8s-test.go ${SRC_DIR}/csv.go ${SRC_DIR}/types.go
	mv k8s-test ${K8S_TEST}
docker:
	go build ${SRC_DIR}/docker-test.go ${SRC_DIR}/csv.go ${SRC_DIR}/types.go
	mv docker-test ${DOCKER_TEST}
clean:
	rm -f ${K8S_API_PREEMPT_TEST}
	rm -f ${K8S_API_TEST}
	rm -f ${K8S_TEST}
	rm -f ${DOCKER_TEST}
