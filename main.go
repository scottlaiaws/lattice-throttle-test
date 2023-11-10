package main

import (
	"fmt"
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
	var create int
	var read int

	var wg sync.WaitGroup

	fmt.Print("Enter create rate per second: ")
	fmt.Scan(&create)
	fmt.Print("Enter read rate per second: ")
	fmt.Scan(&read)
	fmt.Println()

	// runThrottleTest("list service networks", 1000, lattice.listSn)

	for i := 0; i < 60; i++ {
		waitNextSecond()

		if create > 0 {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()
				runThrottleTest("create", create, lattice.createSn)
			}(i)
		}

		if read > 0 {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()
				runThrottleTest("read", read, lattice.listSn)
			}(i)
		}

		wg.Wait()

	}

	fmt.Println("Testing done!")
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
	printResultsSummary(results, name)
	return results
}

type Lattice struct {
	c *VPCLattice
}

func NewLattice() *Lattice {
	cfg := &aws.Config{
		Region:     aws.String("us-east-1"),
		MaxRetries: aws.Int(0),
		Retryer:    &NoRetry{},
	}
	sess, err := session.NewSession(cfg)

	if err != nil {
		panic(err)
	}

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

func printResultsSummary(res []Result, name string) {
	total := len(res)
	success := 0
	throttled := 0
	for _, r := range res {
		if r.err == nil {
			success += 1
			// fmt.Printf("start=%s,end=%s\n", r.start.Format(TimeFormat), r.end.Format(TimeFormat))
		} else {
			code := "unknown"
			if aerr, ok := r.err.(awserr.Error); ok {
				code = aerr.Code()
			}

			if code == "ThrottlingException" {
				throttled += 1
			} else {
				success += 1
			}
		}

	}
	fmt.Printf("| %-10s | total=%03d | throttled=%03d\n", name, total, throttled)
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
