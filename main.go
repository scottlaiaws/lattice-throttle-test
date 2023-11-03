package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	. "github.com/aws/aws-sdk-go/service/vpclattice"
)

func main() {
	lattice := NewLattice()
	waitNextSecond()
	res := runThrottleTest("list sn", 100, lattice.listSn)
	printResults(res)
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

func runThrottleTest(name string, n int, f func() error) []Result {
	log.Printf("starting throttle test, name=%s, n=%d", name, n)
	var wg sync.WaitGroup
	var outLock sync.Mutex
	var out []Result
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			err := f()
			end := time.Now()
			res := Result{
				start: start,
				end:   end,
				err:   err,
			}
			outLock.Lock()
			out = append(out, res)
			outLock.Unlock()
		}()
	}
	wg.Wait()
	log.Printf("finished throttle test, name=%s, n=%d", name, n)
	return out
}

type Lattice struct {
	c *VPCLattice
}

func NewLattice() *Lattice {
	cfg := &aws.Config{
		Region:     aws.String("us-west-2"),
		MaxRetries: aws.Int(0),
	}
	sess := session.New(cfg)
	client := New(sess)
	return &Lattice{c: client}
}

func (l *Lattice) listSn() error {
	_, err := l.c.ListServiceNetworks(&ListServiceNetworksInput{})
	return err
}

func printResults(res []Result) {
	for i, r := range res {
		fmt.Printf("i=%d, r=%s\n", i, r)
	}
}
