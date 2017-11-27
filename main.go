package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/shidenkai0/nscheck/ns"
)

func main() {
	domain := flag.String("d", "google.com", "domain to query")
	rType := flag.String("t", "A", "Record Type to query")
	serversCSV := flag.String("f", "nameservers-all.csv", "DNS Servers list (CSV)")
	nsList, err := ns.Load(*serversCSV)
	if err != nil {
		log.Fatalln(fmt.Errorf("Loading NS List: %v", err))
	}
	queryLogs, err := nsList.Query(*domain, *rType)
	if err != nil {
		panic(err)
	}
	if len(queryLogs) >= 1 { // Print only first NSLog so far
		fmt.Println(queryLogs[0])
	}
}
