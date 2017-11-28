package ns

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/miekg/dns"
)

type Query struct {
	RecordType string
	Name       string
}

// NS represents a name server
type NS struct {
	IP          net.IP    `csv:"ip"`
	Name        string    `csv:"name"`
	CountryID   string    `csv:"country_id"`
	City        string    `csv:"city"`
	Reliability float64   `csv:"reliability"`
	CheckedAt   time.Time `csv:"checked_at"`
	CreatedAt   time.Time `csv:"created_at"`
}

// Query a name server
func (ns *NS) Perform(q Query) ([]string, error) {
	q.Name += "."
	m := new(dns.Msg)
	m.SetQuestion(q.Name, dns.StringToType[q.RecordType])
	//c := new(dns.Client)
	res, err := dns.Exchange(m, fmt.Sprintf("%v:53", ns.IP))
	if err != nil {
		return nil, err
	}
	var answers []string
	for _, rr := range res.Answer {
		answers = append(answers, rr.String())
	}
	return answers, nil
}

type NSQuery struct {
	Ns NS
	Q  Query
}

func (nsq *NSQuery) perform() QueryResult {
	answers, err := nsq.Ns.Perform(nsq.Q)
	return QueryResult{Server: nsq.Ns, Answers: answers, Q: nsq.Q, Err: err}
}

// NSList is a queriable list of NSes
type NSList []NS

func (nsl *NSList) Perform(q Query) <-chan QueryResult {
	return process(nsl.sequentialQueries(q), 256)
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
	Server  NS
	Q       Query
	Err     error
	Answers []string
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
	return process(buildNSQueries(StreamFromCSV(fileName), q), 256)
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
