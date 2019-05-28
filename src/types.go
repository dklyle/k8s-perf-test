package main

// type for returning timing values from go routines via channels
type TimingRecord struct {
	name string
	time int64
}

type NodeTimingRecord struct {
	TimingRecord
	node string
}

// stores a node utilization sample
type NodeUtilizationRecord struct {
	node   string // name of the node sample is for
	cpu    string // percentage value as string 0-100
	memory string // percentage value as string 0-100
	time   int64  // Unix timestamp of sample, nanoseconds
}
