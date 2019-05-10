package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// naming prefix for pods spawned
const POD_NAME_PREFIX = "bb-"

// go routine to invoke pod starts
// returns the TimingRecord of the start time via channel
func runPod(wg *sync.WaitGroup, start chan<- TimingRecord, id int, image string) {
	defer wg.Done()

	// curl -s http://localhost:8090/api/v1/namespaces/default/pods -XPOST -H 'Content-Type: application/json' -d@bb.json > /dev/null
	podName := fmt.Sprintf("%v%v", POD_NAME_PREFIX, id)
	commandString := fmt.Sprintf("curl -s http://localhost:8090/api/v1/namespaces/default/pods -XPOST -H 'Content-Type: application/json' -d@/tmp/%v.json > /dev/null", podName)

	cmd := exec.Command("/bin/sh", "-c", commandString)
	startTime := time.Now()

	err := cmd.Run()
	if err != nil {
		fmt.Println("curl pod POST failed for " + podName)
		fmt.Println(err)
	}

	var s TimingRecord
	s.name = podName
	s.time = startTime.UnixNano()

	start <- s
}

func terminatePod(wg *sync.WaitGroup, ended chan<- TimingRecord, name string, grace int) {
	defer wg.Done()

	// curl -s http://localhost:8090/api/v1/namespaces/default/pods/$name?gracePeriodSeconds=$grace -XDELETE -H 'Content-Type: application/json'
	commandString := fmt.Sprintf("curl -s http://localhost:8090/api/v1/namespaces/default/pods/%v?gracePeriodSeconds=%v -XDELETE -H 'Content-Type: application/json'", name, grace)
	cmd := exec.Command("/bin/sh", "-c", commandString)

	err := cmd.Run()
	if err != nil {
		fmt.Printf("curl pod DELETE failed for %v\n", name)
		fmt.Println(err)
	}

	e := pollPodTermination(name)

	ended <- e
}

// function for watching for the pod to get fully cleaned up
// i.e., the API returns 404 when we do a GET on the pod
//
// This is heavier than I would like as we're polling for each pod, rather
// than collectively for all pods. Need to figure out a cleaner way, or just
// use kubectl
func pollPodTermination(name string) TimingRecord {
	// curl -s http://localhost:8090/api/v1/namespaces/default/pod/bb-0 -XGET -H 'Content-Type: application/json'
	commandString := fmt.Sprintf("curl -s http://localhost:8090/api/v1/namespaces/default/pods/%v -XGET -H 'Content-Type: application/json'", name)

	// loop until the pod is gone
	for {
		cmd := exec.Command("/bin/sh", "-c", commandString)
		out, err := cmd.Output()
		if err != nil {
			fmt.Println("curl pod GET failed for " + name)
			fmt.Println(err)
		}
		if strings.Contains(string(out), "\"code\": 404") {
			break
		}

		// polling interval
		time.Sleep(200 * time.Millisecond)
	}

	endTime := time.Now()

	var e TimingRecord
	e.name = name
	e.time = endTime.UnixNano()

	return e
}

// function for detecting running pods
func findRunningPod(wg *sync.WaitGroup, running chan<- TimingRecord, ending chan<- TimingRecord, pods, grace int) {
	defer wg.Done()

	// wait group for pod termination calls
	var termWG sync.WaitGroup
	// channel for receiving end times of termination calls
	ended := make(chan TimingRecord)

	// map for storing time when pod reaches is fully deleted
	endTimes := make(map[string]int64)

	// map for storing time when pod reaches running state
	runningTimes := make(map[string]int64)

	// poll kubectl until all expected pods have reached the running state
	// once a pod reaches the running state, a goroutine to terminate the
	// pod is started
	for len(runningTimes) < pods {
		// show all pod names in running state
		// kubectl get pods --no-headers --field-selector=status.phase=Running | awk {'print $1'}
		commandString := "kubectl get pods --no-headers --field-selector=status.phase=Running | awk {'print $1'}"
		cmd := exec.Command("/bin/sh", "-c", commandString)

		// record the time the command is run
		now := time.Now()
		out, err := cmd.Output()
		if err != nil {
			fmt.Println("kubectl get pods failed")
			fmt.Println(err)
		}
		// each pod in the running state is output on a newline
		results := strings.Split(string(out), "\n")
		for _, r := range results {
			// above split results contains an empty line, so eliminating those
			// other pods may be running on the system, only kill the desired
			// started by this test
			if len(r) > 0 && strings.HasPrefix(r, POD_NAME_PREFIX) {
				// if we haven't seen this pod name in the running state,
				// record the time
				if _, ok := runningTimes[r]; !ok {
					runningTimes[r] = now.UnixNano()

					// now call kubectl to stop the pod
					termWG.Add(1)
					go terminatePod(&termWG, ended, r, grace)
				}
			}
		}

		// polling interval
		time.Sleep(100 * time.Millisecond)
	}

	// goroutine to receive end time from termination go routines
	go func() {
		for e := range ended {
			endTimes[e.name] = e.time
		}
	}()

	// wait for all termination jobs to complete
	termWG.Wait()

	// extra time to allow channel values to be read
	time.Sleep(3000 * time.Millisecond)

	// return the results over the running channel
	for key, value := range runningTimes {
		var rRecord TimingRecord
		rRecord.name = key
		rRecord.time = value
		running <- rRecord
	}

	// return the results over the ending channel
	for key, value := range endTimes {
		var eRecord TimingRecord
		eRecord.name = key
		eRecord.time = value
		ending <- eRecord
	}
}

// type for returning timing values from go routines via channels
type TimingRecord struct {
	name string
	time int64
}

type PodJson struct {
	ApiVersion string      `json:"apiVersion"`
	Kind       string      `json:"kind"`
	Metadata   PodMetadata `json:"metadata"`
	Spec       PodSpec     `json:"spec"`
}

type PodMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type PodSpec struct {
	Containers    []PodSpecContainer `json:"containers"`
	RestartPolicy string             `json:"restartPolicy"`
}

type PodSpecContainer struct {
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Command []string `json:"command"`
}

// creates 'pods' number of json files in /tmp
func createPodJsonFiles(pods int, imageName string) {
	for i := 0; i < pods; i++ {
		podName := fmt.Sprintf("%v%v", POD_NAME_PREFIX, i)
		jsonFileName := fmt.Sprintf("/tmp/%v.json", podName)

		// if file does not already exist, create it, otherwise skip create
		if _, err := os.Stat(jsonFileName); os.IsNotExist(err) {
			podSpecContainer := PodSpecContainer{
				Name:    imageName,
				Image:   imageName,
				Command: []string{"sleep", "3600"},
			}
			podSpec := PodSpec{
				Containers:    []PodSpecContainer{podSpecContainer},
				RestartPolicy: "Never",
			}
			podMetadata := PodMetadata{
				Name:      podName,
				Namespace: "default",
			}
			podJson := &PodJson{
				ApiVersion: "v1",
				Kind:       "Pod",
				Metadata:   podMetadata,
				Spec:       podSpec,
			}

			// convert data structure to json
			var jsonData []byte
			jsonData, err := json.MarshalIndent(podJson, "", "  ")
			if err != nil {
				fmt.Println("Json marshal failed for " + podName)
				panic(err)
			}

			// write the file
			err = ioutil.WriteFile(jsonFileName, jsonData, 0644)
			if err != nil {
				fmt.Println("Error writing file " + jsonFileName)
				panic(err)
			}
		}
	}
}

func main() {

	// input for the number of pods to use for this test run
	numPtr := flag.Int("num", 1, "the number of pods to launch")

	// the amount of time between SIGTERM and SIGKILL for pod termination
	// the default for kubectl delete pod is 30 seconds, preserving that default
	gracePtr := flag.Int("grace", 30, "the number of seconds for graceful shutdown")

	csvPtr := flag.Bool("csv", false, "write results to CSV format file")

	flag.Parse()

	imageName := "busybox"

	// allocate map for start times
	// assumes time was stored as time.Now().UnixNano() which returns int64
	startTimes := make(map[string]int64)
	// set up start time channel to collect the start times from the go routines
	starts := make(chan TimingRecord)

	// allocate map for running times
	runningTimes := make(map[string]int64)
	// assumes time was stored as time.Now().UnixNano()
	running := make(chan TimingRecord)

	// allocate map for end times
	// assumes time was stored as time.Now().UnixNano()
	endTimes := make(map[string]int64)
	// assumes time was stored as time.Now().UnixNano()
	ended := make(chan TimingRecord)

	var wg sync.WaitGroup

	fmt.Printf("Running test with %v pods and a shutdown grace period of %v seconds\n", *numPtr, *gracePtr)

	// check/create missing json files for pod specifications
	createPodJsonFiles(*numPtr, imageName)

	// go routine to poll for "running" times
	wg.Add(1)
	go findRunningPod(&wg, running, ended, *numPtr, *gracePtr)

	// go routine for each start
	// receive start time from channel and record
	for i := 0; i < *numPtr; i++ {
		wg.Add(1)
		go runPod(&wg, starts, i, "busybox")
	}

	// goroutine to wait receive start times from runPod goroutines
	go func() {
		for s := range starts {
			startTimes[s.name] = s.time
		}
	}()

	// goroutine to wait receive running times from findRunningPod goroutine
	go func() {
		for r := range running {
			runningTimes[r.name] = r.time
		}
	}()

	// goroutine to wait receive end times from findRunningPod goroutine
	go func() {
		for e := range ended {
			endTimes[e.name] = e.time
		}
	}()

	// wait for all the goroutines to return
	wg.Wait()

	// sleep to allow channel buffers to clear
	time.Sleep(3000 * time.Millisecond)

	// print results to console
	for key, value := range startTimes {
		fmt.Printf("%v: start: %v\trunning: %v\tterminated :%v\n", key, value, runningTimes[key], endTimes[key])
		fmt.Printf("\t%v milliseconds from start to running\n", (runningTimes[key]-value)/int64(time.Millisecond))
		fmt.Printf("\t%v milliseconds from running to terminated\n", (endTimes[key]-runningTimes[key])/int64(time.Millisecond))
	}

	if *csvPtr == true {
		writeCSV(*numPtr, *gracePtr, startTimes, runningTimes, endTimes)
	}
}