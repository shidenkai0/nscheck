package main

import (
	"flag"
	"fmt"

	"github.com/shidenkai0/nscheck/ns"
)

func main() {
	name := flag.String("n", "google.com", "DNS name to query")
	rType := flag.String("t", "A", "Record Type to query")
	serversCSV := flag.String("f", "nameservers-all.csv", "DNS Servers list (CSV)")
	flag.Parse()

	queryLogsChan := ns.PerformFromCSV(ns.Query{RecordType: *rType, Name: *name}, *serversCSV)
	var successCount int
	for ql := range queryLogsChan {
		if ql.Err == nil {
			successCount++
		}
	}
	fmt.Println("Successful queries:", successCount)
}
