package main

import (
	"flag"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// number of seconds for container to sleep
const CONTAINER_SLEEP = 200

// naming prefix for containers spawned
const CONTAINER_NAME_PREFIX = "bb-"

// go routine to invoke container starts
// returns the TimingRecord of the start time via channel
func runContainer(wg *sync.WaitGroup, start chan<- TimingRecord, id int, image string) {
	defer wg.Done()

	// docker run --rm --name bb-$id image sh -c 'exec sleep 200' &
	containerName := fmt.Sprintf("%v%v", CONTAINER_NAME_PREFIX, id)
	commandString := fmt.Sprintf("docker run --rm --name %v %v sh -c 'exec sleep %v' &", containerName, image, CONTAINER_SLEEP)

	cmd := exec.Command("/bin/sh", "-c", commandString)
	startTime := time.Now()

	err := cmd.Run()
	if err != nil {
		fmt.Println("docker run failed for " + containerName)
		fmt.Println(err)
	}

	var s TimingRecord
	s.name = containerName
	s.time = startTime.UnixNano()

	start <- s
}

func terminateContainer(wg *sync.WaitGroup, ended chan<- TimingRecord, name string, grace int) {
	defer wg.Done()

	commandString := fmt.Sprintf("docker stop %v --time %v", name, grace)
	cmd := exec.Command("/bin/sh", "-c", commandString)

	err := cmd.Run()
	if err != nil {
		fmt.Printf("docker stop failed for %v\n", name)
		fmt.Println(err)
	}

	endTime := time.Now()

	var e TimingRecord
	e.name = name
	e.time = endTime.UnixNano()

	ended <- e
}

// command to get a status for a specific container
// docker inspect --format='{{.State.Status}}' bb-1

// function for detecting running containers
func findRunningContainers(wg *sync.WaitGroup, running chan<- TimingRecord, ending chan<- TimingRecord, containers, grace int) {
	defer wg.Done()

	// wait group for container termination calls
	var termWG sync.WaitGroup
	// channel for receiving end times of termination calls
	ended := make(chan TimingRecord)

	// map for storing time when container reaches is fully deleted
	endTimes := make(map[string]int64)

	// map for storing time when container reaches running state
	runningTimes := make(map[string]int64)

	// poll docker daemon until all expected containers have reached the running state
	// once a container reaches the running state, a goroutine to terminate the
	// container is started
	for len(runningTimes) < containers {
		// show all container names in running state
		// docker ps --filter status=running --format {{.Names}}
		commandString := "docker ps --filter status=running --format {{.Names}}"
		cmd := exec.Command("/bin/sh", "-c", commandString)

		// record the time the command is run
		now := time.Now()
		out, err := cmd.Output()
		if err != nil {
			fmt.Println("docker ps failed")
			fmt.Println(err)
		}
		// each container in the running state is output on a newline
		results := strings.Split(string(out), "\n")
		for _, r := range results {
			// above split contains an empty line, so eliminating those
			// other containers may be running on the system, only track and
			// delete containers spawned in this test
			if len(r) > 0 && strings.HasPrefix(r, CONTAINER_NAME_PREFIX) {
				// if we haven't seen this container name in the running state,
				// record the time
				if _, ok := runningTimes[r]; !ok {
					runningTimes[r] = now.UnixNano()

					// now call docker to stop the container
					termWG.Add(1)
					go terminateContainer(&termWG, ended, r, grace)
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

func main() {

	// input for the number of containers to use for this test run
	numPtr := flag.Int("num", 1, "the number of docker containers to launch")

	// the amount of time between SIGTERM and SIGKILL for container termination
	// the default for docker stop is 10 seconds, preserving that default
	gracePtr := flag.Int("grace", 10, "the number of seconds for graceful shutdown")

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

	fmt.Printf("Running test with %v containers and a shutdown grace period of %v seconds\n", *numPtr, *gracePtr)

	// go routine to poll for "running" times
	wg.Add(1)
	go findRunningContainers(&wg, running, ended, *numPtr, *gracePtr)

	// go routine for each start
	// receive start time from channel and record
	for i := 0; i < *numPtr; i++ {
		wg.Add(1)
		go runContainer(&wg, starts, i, "busybox")
	}

	// goroutine to wait receive start times from runContainer goroutines
	go func() {
		for s := range starts {
			startTimes[s.name] = s.time
		}
	}()

	// goroutine to wait receive running times from findRunningContainer goroutine
	go func() {
		for r := range running {
			runningTimes[r.name] = r.time
		}
	}()

	// goroutine to wait receive end times from findRunningContainer goroutine
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
		fmt.Printf("%v: start: %v\trunning: %v\tterminated :%v\n", key, value, runningTimes[key], endTimes[key])
		fmt.Printf("\t%v milliseconds from start to running\n", (runningTimes[key]-value)/int64(time.Millisecond))
		fmt.Printf("\t%v milliseconds from running to terminated\n", (endTimes[key]-runningTimes[key])/int64(time.Millisecond))
	}

	if *csvPtr == true {
		writeCSV(*numPtr, *gracePtr, startTimes, runningTimes, endTimes)
	}
}
