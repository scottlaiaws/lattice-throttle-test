package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	. "github.com/aws/aws-sdk-go/service/vpclattice"
)

func main() {
	lattice := NewLattice()

	// waitNextSecond()
	// results := runThrottleTest("list sn", 1000, lattice.listSn)
	// printResultsSummary(results)

	// waitNextSecond()
	// results := runThrottleTest("list svcs", 1000, lattice.listSvc)
	// printResultsSummary(results)

	waitNextSecond()
	runThrottleTest("create sn", 20, lattice.createSn)
}

// waits till beginning of next second
func waitNextSecond() {
	now := time.Now()
	next := now.Truncate(time.Second).Add(time.Second)
	time.Sleep(next.Sub(now))
}

type Result struct {
	start time.Time
	end   time.Time
	err   error
}

const TimeFormat = "15:04:05.000"

func (r Result) String() string {
	return fmt.Sprintf("success=%t, err=%s, start=%s, stop=%s",
		r.err == nil, r.err, r.start.Format(TimeFormat), r.end.Format(TimeFormat))
}

func runThrottleTest(name string, concurrency int, f func() error) []Result {
	log.Printf("starting throttle test, name=%s, n=%d", name, concurrency)
	var wg sync.WaitGroup
	var outLock sync.Mutex
	var results []Result
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			err := f()
			end := time.Now()
			result := Result{
				start: start,
				end:   end,
				err:   err,
			}
			outLock.Lock()
			results = append(results, result)
			outLock.Unlock()
		}()
	}
	wg.Wait()
	printResultsSummary(results)
	return results
}

type Lattice struct {
	c *VPCLattice
}

func NewLattice() *Lattice {
	cfg := &aws.Config{
		Region:     aws.String("us-west-2"),
		MaxRetries: aws.Int(0),
		Retryer:    &NoRetry{},
	}
	sess := session.New(cfg)
	client := New(sess)
	return &Lattice{c: client}
}

func (l *Lattice) listSn() error {
	_, err := l.c.ListServiceNetworks(&ListServiceNetworksInput{})
	return err
}

func (l *Lattice) listSvc() error {
	_, err := l.c.ListServices(&ListServicesInput{})
	return err
}

func (l *Lattice) createSn() error {
	id := rand.Int()
	snName := fmt.Sprintf("%d-throttle-test", id)
	_, err := l.c.CreateServiceNetwork(&CreateServiceNetworkInput{
		Name: &snName,
	})
	return err
}

func printResults(res []Result) {
	for i, r := range res {
		fmt.Printf("i=%d, r=%s\n", i, r)
	}
}

func printResultsSummary(res []Result) {
	total := len(res)
	success := 0
	errors := map[string]int{}
	for _, r := range res {
		if r.err == nil {
			success += 1
		} else {
			code := "unknown"
			if aerr, ok := r.err.(awserr.Error); ok {
				code = aerr.Code()
			}
			errCnt := errors[code]
			errCnt += 1
			errors[code] = errCnt
		}
	}
	log.Printf("results summary, total=%d, success=%d, errors=%v", total, success, errors)
}

type NoRetry struct {
}

func (r *NoRetry) RetryRules(_ *request.Request) time.Duration {
	return 0
}

func (r *NoRetry) ShouldRetry(_ *request.Request) bool {
	return false
}

func (r *NoRetry) MaxRetries() int {
	return 0
}
