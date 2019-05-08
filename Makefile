all: k8s-api-test k8s-test docker-test
k8s-api-test:
	go build k8s-api-test.go csv.go
k8s-test:
	go build k8s-test.go csv.go
docker-test:
	go build docker-test.go csv.go
clean:
	rm -f k8s-api-test
	rm -f k8s-test
	rm -f docker-test
