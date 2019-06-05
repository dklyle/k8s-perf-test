package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// type to handle map values when sorting
type KeyValue struct {
	Key   string
	Value int64
}

func sortByStartTime(startTimes map[string]int64) []KeyValue {
	var list []KeyValue
	// convert format to allow sorting
	for k, v := range startTimes {
		list = append(list, KeyValue{k, v})
	}

	// sort list in ascending order
	sort.Slice(list, func(i, j int) bool {
		return list[i].Value < list[j].Value
	})

	return list
}

func genFilePathAndName(num, grace int) (string, string) {
	// build filename
	programName := os.Args[0]
	// strip local dir invocation if present
	if strings.HasPrefix(os.Args[0], "./") {
		programName = os.Args[0][2:]
	}
	// create a directory with the current timestamp
	now := time.Now().UnixNano()
	directoryName := strconv.FormatInt(now, 10)
	// since the directory is a timestamp, not error checking for existence
	os.Mkdir(directoryName, 0755)

	// file format encodes number, grace period, and what program ran the test
	fileName := fmt.Sprintf("n%v-g%v-%s.csv", num, grace, programName)
	filePath := filepath.Join(directoryName, fileName)

	return filePath, fileName
}

func writeCSV(num, grace int, startTimes, runningTimes, endTimes map[string]int64) {
	filePath, fileName := genFilePathAndName(num, grace)

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error create csv file: ", fileName)
		fmt.Println(err)
		return
	}

	// sort the start times
	starts := sortByStartTime(startTimes)

	// some place to store the csv rows to write
	rows := make([][]string, num)
	for i, start := range starts {
		rows[i] = []string{
			strconv.Itoa(i),
			strconv.FormatInt((runningTimes[start.Key]-start.Value)/int64(time.Millisecond), 10),
			strconv.FormatInt((endTimes[start.Key]-runningTimes[start.Key])/int64(time.Millisecond), 10),
		}
	}

	// write the csv file, WriteAll handles the flush
	err = csv.NewWriter(file).WriteAll(rows)
	if err != nil {
		fmt.Println("Failed to write csv file: ", fileName)
		fmt.Println(err)
	}
	file.Close()
}

// find the interpolated utilization record
func interpolateUtilizationRecord(node string, rTime int64, nodes map[string][]NodeUtilizationRecord) NodeUtilizationRecord {
	var utilRecord NodeUtilizationRecord

	// store value for timestamp less than timestamp
	var previous NodeUtilizationRecord
	// store value for timestamp greater than timestamp
	var next NodeUtilizationRecord

	// grab the utilization on the node this pod is running on
	for _, value := range nodes[node] {
		if value.time <= rTime {
			previous = value
		}
		if value.time >= rTime {
			next = value
			// if we found the next timestamp value, stop iterating
			break
		}
	}
	// calculate weights for weighted average
	prevDiff := rTime - previous.time
	nextDiff := next.time - rTime
	totalDiff := next.time - previous.time

	if totalDiff == 0 {
		// set value to prev as the records are the same
		utilRecord = previous
	} else {
		prevCpuInt, _ := strconv.Atoi(previous.cpu)
		prevMemInt, _ := strconv.Atoi(previous.memory)
		nextCpuInt, _ := strconv.Atoi(next.cpu)
		nextMemInt, _ := strconv.Atoi(next.memory)
		prevWeight := float64(prevDiff) / float64(totalDiff)
		nextWeight := float64(nextDiff) / float64(totalDiff)
		var r NodeUtilizationRecord
		r.node = previous.node
		r.time = rTime
		// calculate the weighted average i.e., lazy interpolation
		r.cpu = strconv.FormatFloat((prevWeight*float64(prevCpuInt) + nextWeight*float64(nextCpuInt)), 'f', 0, 64)
		r.memory = strconv.FormatFloat((prevWeight*float64(prevMemInt) + nextWeight*float64(nextMemInt)), 'f', 0, 64)
		utilRecord = r
	}
	return utilRecord
}

func writeNodeRecordTimingCSV(num, grace int, startTimes, runningTimes map[string]int64, endTimes map[string]NodeTimingRecord, nodeRecords map[string][]NodeUtilizationRecord) {
	filePath, fileName := genFilePathAndName(num, grace)

	// sort per node records by timestamp
	nodes := make(map[string][]NodeUtilizationRecord)
	for nodeName, records := range nodeRecords {
		sortedRecords := sortNodeRecordsByTimestamp(records)
		nodes[nodeName] = sortedRecords
	}

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error create csv file: ", fileName)
		fmt.Println(err)
		return
	}

	// sort the start times
	starts := sortByStartTime(startTimes)

	// some place to store the csv rows to write
	rows := make([][]string, num)

	// build the rows
	for i, start := range starts {
		utilRecord := interpolateUtilizationRecord(endTimes[start.Key].node, runningTimes[start.Key], nodes)
		// this is a kludge to record master node utilization, hard coded to 1 master
		// currently, will need to come up with something more programmatic when time
		// allows
		masterRecord := interpolateUtilizationRecord("node1", runningTimes[start.Key], nodes)
		rows[i] = []string{
			strconv.Itoa(i),
			strconv.FormatInt((runningTimes[start.Key]-start.Value)/int64(time.Millisecond), 10),
			strconv.FormatInt((endTimes[start.Key].time-runningTimes[start.Key])/int64(time.Millisecond), 10),
			endTimes[start.Key].node,
			utilRecord.cpu,
			utilRecord.memory,
			masterRecord.cpu,
			masterRecord.memory,
		}
	}

	// write the csv file, WriteAll handles the flush
	err = csv.NewWriter(file).WriteAll(rows)
	if err != nil {
		fmt.Println("Failed to write csv file: ", fileName)
		fmt.Println(err)
	}
	file.Close()
}

func sortNodeRecordsByTimestamp(records []NodeUtilizationRecord) []NodeUtilizationRecord {
	// sort records in chronological (ascending) order
	sort.Slice(records, func(i, j int) bool {
		return records[i].time < records[j].time
	})

	return records
}

// num and grace parameters are only for file naming
func writeNodeUtilizationCSV(nodeRecords map[string][]NodeUtilizationRecord, num, grace int) {
	i := 0
	// iterate over the nodes
	for _, v := range nodeRecords {
		// sort the records by timestamp
		records := sortNodeRecordsByTimestamp(v)

		// build filename
		programName := os.Args[0]
		// strip local dir invocation if present
		if strings.LastIndex(os.Args[0], "/") != -1 {
			i := strings.LastIndex(os.Args[0], "/")
			programName = os.Args[0][(i + 1):]
		}
		// create a directory with the current timestamp
		now := strconv.FormatInt(records[0].time, 10)
		directoryName := fmt.Sprintf("%s-%s", records[0].node, now)
		// since the directory is a timestamp, not error checking for existence
		os.Mkdir(directoryName, 0755)

		// file format encodes number, grace period, and what program ran the test
		fileName := fmt.Sprintf("n%d-g%d-%s-%s-util.csv", num, grace, records[0].node, programName)
		filePath := filepath.Join(directoryName, fileName)

		file, err := os.Create(filePath)
		if err != nil {
			fmt.Println("Error create csv file: ", fileName)
			fmt.Println(err)
		}

		rows := make([][]string, len(records))
		for i, r := range records {
			rows[i] = []string{
				strconv.FormatInt(r.time/int64(time.Millisecond), 10), // converting ns timestamp to ms
				r.cpu,
				r.memory,
			}
		}

		// write the csv file, WriteAll handles the flush
		err = csv.NewWriter(file).WriteAll(rows)
		if err != nil {
			fmt.Println("Failed to write csv file: ", fileName)
			fmt.Println(err)
		}
		file.Close()

		i++
	}
}
