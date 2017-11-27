package ns

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/miekg/dns"
)

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
func (ns *NS) Query(domain, recordType string) ([]string, error) {
	m := new(dns.Msg)
	m.SetQuestion(domain, dns.StringToType[recordType])
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

// NSList is a queriable list of NSes
type NSList []NS

func (nsl *NSList) Query(domain, recordType string) ([]NSLog, error) {
	if nsl == nil {
		return nil, fmt.Errorf("ns: Query on nil NSList")
	}
	var qls []NSLog // query logs
	for _, ns := range *nsl {
		answers, err := ns.Query(domain, recordType)
		ql := NSLog{
			Server:  ns,
			Err:     err,
			Answers: answers,
		}
		qls = append(qls, ql)
	}
	return qls, nil
}

// NSLog represents a set of DNS answers along with metadata on the associated name server
type NSLog struct {
	Server  NS
	Err     error
	Answers []string
}

// Load loads a DNS list from CSV
func Load(fileName string) (NSList, error) {
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
