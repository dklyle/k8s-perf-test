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

func writeCSV(num, grace int, startTimes, runningTimes, endTimes map[string]int64) {
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
