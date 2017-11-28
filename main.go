package main

import (
	"flag"
	"fmt"

	"github.com/miekg/dns"

	"github.com/shidenkai0/nscheck/ns"
)

func main() {
	name := flag.String("n", "google.com", "DNS name to query")
	rType := flag.String("t", "A", "Record Type to query")
	serversCSV := flag.String("f", "nameservers-all.csv", "DNS Servers list (CSV)")
	flag.Parse()

	queryLogsChan := ns.PerformFromCSV(ns.Query{RecordType: *rType, Name: *name}, *serversCSV)
	rrs := make(map[string]int)
	var queriesCount, successCount int
	for ql := range queryLogsChan {
		queriesCount++
		if ql.Err != nil {
			continue
		}
		successCount++
		for _, rr := range ql.RR {
			switch v := rr.(type) {
			case *dns.CNAME:
				rrs[dns.TypeToString[v.Header().Rrtype]+" "+v.Target]++
			case *dns.A:
				rrs[dns.TypeToString[v.Header().Rrtype]+" "+v.A.String()]++
			}
		}
	}
	fmt.Printf("Returned Records:\n")
	for i, v := range rrs {
		fmt.Printf("%v: %v\n", i, v)
	}
	fmt.Printf("Successful queried %d out of %d servers.\n", successCount, queriesCount)
}
