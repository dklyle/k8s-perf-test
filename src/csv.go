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

func writeNodeRecordTimingCSV(num, grace int, startTimes, runningTimes map[string]int64, endTimes map[string]NodeTimingRecord) {
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
			strconv.FormatInt((endTimes[start.Key].time-runningTimes[start.Key])/int64(time.Millisecond), 10),
			endTimes[start.Key].node,
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
		if strings.HasPrefix(os.Args[0], "./") {
			programName = os.Args[0][2:]
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
