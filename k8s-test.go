package main

import (
	"flag"
	"fmt"
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

	// kubectl run -i --tty bb-$id image -- sh &
	podName := fmt.Sprintf("%v%v", POD_NAME_PREFIX, id)
	commandString := fmt.Sprintf("kubectl run -i --tty %v --image=%v --restart=Never -- sh &", podName, image)

	cmd := exec.Command("/bin/sh", "-c", commandString)
	startTime := time.Now()

	err := cmd.Run()
	if err != nil {
		fmt.Println("kubectl run failed for " + podName)
		fmt.Println(err)
	}

	var s TimingRecord
	s.name = podName
	s.time = startTime.UnixNano()

	start <- s
}

func terminatePod(wg *sync.WaitGroup, ended chan<- TimingRecord, name string, grace int) {
	defer wg.Done()

	// kubectl delete pod $name --grace-period=$grace
	commandString := fmt.Sprintf("kubectl delete pod %v --grace-period=%v", name, grace)
	cmd := exec.Command("/bin/sh", "-c", commandString)

	err := cmd.Run()
	if err != nil {
		fmt.Printf("kubectl delete pod failed for %v\n", name)
		fmt.Println(err)
	}

	endTime := time.Now()

	var e TimingRecord
	e.name = name
	e.time = endTime.UnixNano()

	ended <- e
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
		fmt.Println("find")
		fmt.Println(results)
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

func main() {

	// input for the number of pods to use for this test run
	numPtr := flag.Int("num", 1, "the number of pods to launch")

	// the amount of time between SIGTERM and SIGKILL for pod termination
	// the default for kubectl delete pod is 30 seconds, preserving that default
	gracePtr := flag.Int("grace", 30, "the number of seconds for graceful shutdown")

	csvPtr := flag.Bool("csv", false, "write results to CSV format file")

	flag.Parse()

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
	for key, value := range startTimes {
		fmt.Printf("start %v:%v:%v:%v\n", key, value, runningTimes[key], endTimes[key])
		fmt.Printf("\t%v\n", (runningTimes[key]-value)/int64(time.Millisecond))
		fmt.Printf("\t%v\n", (endTimes[key]-runningTimes[key])/int64(time.Millisecond))
	}

	if *csvPtr == true {
		writeCSV(*numPtr, *gracePtr, startTimes, runningTimes, endTimes)
	}
}
