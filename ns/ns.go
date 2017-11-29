package ns

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/miekg/dns"
)

var (
	rate         = time.Second / 100
	rateLimiter  = time.Tick(rate)
	ErrInvalidNS = errors.New("ns: name server not functional")
)

type Query struct {
	RecordType string
	Name       string
}

// NS represents a name server
type NS struct {
	IP          string  `csv:"ip"`
	Name        string  `csv:"name"`
	CountryID   string  `csv:"country_id"`
	City        string  `csv:"city"`
	Reliability float64 `csv:"reliability"`
}

// Query a name server
func (ns *NS) Perform(q Query) ([]dns.RR, error) {
	q.Name += "."
	m := new(dns.Msg)
	m.SetQuestion(q.Name, dns.StringToType[q.RecordType])

	retryCount := 3
	var err error
	for i := 0; i < retryCount; i++ {
		var res *dns.Msg
		res, err = dns.Exchange(m, fmt.Sprintf("%s:53", ns.IP))
		if err == nil && len(res.Answer) != 0 {
			return res.Answer, nil
		}
	}
	return nil, err
}

// Check if a name server is valid
func (ns *NS) Check() bool {
	retryCount := 5
	for i := 0; i < retryCount; i++ {
		rrs, err := ns.Perform(Query{Name: "www.amazon.com", RecordType: "A"})
		if err == nil && len(rrs) != 0 {
			return true
		}
	}
	return false
}

type NSQuery struct {
	Ns NS
	Q  Query
}

func (nsq *NSQuery) perform() QueryResult {
	if !nsq.Ns.Check() {
		return QueryResult{Server: nsq.Ns, RR: nil, Q: nsq.Q, Err: ErrInvalidNS}
	}
	rr, err := nsq.Ns.Perform(nsq.Q)
	return QueryResult{Server: nsq.Ns, RR: rr, Q: nsq.Q, Err: err}
}

// NSList is a queriable list of NSes
type NSList []NS

func (nsl *NSList) Perform(q Query) <-chan QueryResult {
	return process(nsl.sequentialQueries(q), 64)
}

func (nsl *NSList) sequentialQueries(q Query) <-chan NSQuery {
	out := make(chan NSQuery)
	go func() {
		for _, ns := range *nsl {
			out <- NSQuery{Ns: ns, Q: q}
		}
		close(out)
	}()
	return out
}

func process(in <-chan NSQuery, concurrency int) <-chan QueryResult {
	out := make(chan QueryResult)
	wg := sync.WaitGroup{}
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			for nsq := range in {
				<-rateLimiter
				ql := nsq.perform()
				out <- ql
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

// QueryResult represents a set of DNS answers along with metadata on the associated name server
type QueryResult struct {
	Server NS
	Q      Query
	Err    error
	RR     []dns.RR
}

func buildNSQueries(in <-chan NS, q Query) <-chan NSQuery {
	out := make(chan NSQuery)
	go func() {
		for ns := range in {
			out <- NSQuery{Q: q, Ns: ns}
		}
		close(out)
	}()
	return out
}

func StreamFromCSV(fileName string) <-chan NS {
	out := make(chan NS)
	nsfile, err := os.Open(fileName)
	if err != nil {
		return nil
	}

	go func() {
		if err := gocsv.UnmarshalToChan(nsfile, out); err != nil {
			fmt.Println(err)
		}
		nsfile.Close()
	}()
	return out
}

func PerformFromCSV(q Query, fileName string) <-chan QueryResult {
	return process(buildNSQueries(StreamFromCSV(fileName), q), 64)
}

// LoadFromCSV loads a DNS list from CSV
func LoadFromCSV(fileName string) (NSList, error) {
	nsfile, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("ns.Load: %v", err)
	}
	defer nsfile.Close()
	nsl := []*NS{}
	if err := gocsv.UnmarshalFile(nsfile, &nsl); err != nil {
		return nil, fmt.Errorf("ns.Load: %v", err)
	}
	var l NSList
	for _, ns := range nsl {
		l = append(l, *ns)
	}
	return l, nil
}
