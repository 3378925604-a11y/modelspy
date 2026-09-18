package probe

import (
	"context"
	"fmt"
	"time"

	"modelspy/internal/client"
	"modelspy/internal/db"
)

type KV struct {
	Key   string
	Value string
}

type Result struct {
	Probe        string
	Description  string
	Skipped      bool
	Error        string
	Observations []KV
	Votes        map[string]float64
	Notes        []string
	Duration     time.Duration
}

type Ctx struct {
	C          *client.Client
	Model      string
	Expect     string
	ExpectCand *db.Candidate
	DB         *db.DB
}

type Probe interface {
	Name() string
	Description() string
	Run(ctx context.Context, x *Ctx) Result
}

func newResult(p Probe) Result {
	return Result{Probe: p.Name(), Description: p.Description(), Votes: map[string]float64{}}
}

func RunAll(parent context.Context, x *Ctx, probes []Probe, progress func(name string, r Result)) []Result {
	results := make([]Result, 0, len(probes))
	for _, p := range probes {
		start := time.Now()
		ctx, cancel := context.WithTimeout(parent, 4*time.Minute)
		r := safeRun(p, ctx, x)
		cancel()
		r.Duration = time.Since(start)
		results = append(results, r)
		if progress != nil {
			progress(p.Name(), r)
		}
	}
	return results
}

func safeRun(p Probe, ctx context.Context, x *Ctx) (r Result) {
	defer func() {
		if e := recover(); e != nil {
			r = newResult(p)
			r.Error = fmt.Sprintf("panic: %v", e)
		}
	}()
	r = p.Run(ctx, x)
	if r.Votes == nil {
		r.Votes = map[string]float64{}
	}
	return r
}

func observe(r *Result, k, v string) {
	r.Observations = append(r.Observations, KV{Key: k, Value: v})
}

func vote(r *Result, candidate string, w float64) {
	r.Votes[candidate] += w
}

func inRange(v float64, rng [2]float64) bool {
	return v >= rng[0] && v <= rng[1]
}
