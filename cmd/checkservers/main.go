package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/gocarina/gocsv"
	"github.com/shidenkai0/nscheck/ns"
)

func main() {
	inputName := flag.String("i", "nameservers-all.csv", "Input NS file to filter")
	outputName := flag.String("o", "nameservers-verified.csv", "Checked servers output")
	flag.Parse()
	nsChan := ns.StreamFromCSV(*inputName)
	if _, err := os.Stat(*outputName); !os.IsNotExist(err) {
		log.Fatalf("Output file %s already exists", *outputName)
	}
	outputFile, err := os.Create(*outputName)
	if err != nil {
		log.Fatalf("Couldn't create output file: %v", err)
	}
	defer outputFile.Close()
	writer := gocsv.NewSafeCSVWriter(csv.NewWriter(outputFile))
	validNSChan := checkNameServers(nsChan, 256)
	var validCounter int
	c := make(chan interface{})
	go func() {
		for Ns := range validNSChan {
			c <- Ns
			validCounter++
		}
		close(c)
	}()
	gocsv.MarshalChan(c, writer)
	fmt.Printf("Completely checked input file.\nNumber of valid servers: %d\n", validCounter)
}

func checkNameServers(in <-chan ns.NS, concurrency int) <-chan ns.NS {
	out := make(chan ns.NS)
	wg := sync.WaitGroup{}
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			for ns := range in {
				if valid := ns.Check(); valid {
					out <- ns
					//gocsv.MarshalFile(ns, outputFile)
				}
			}
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
