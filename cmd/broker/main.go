// broker — the nervous system between the wire and the brains.
//
// The wire (http-mcp) is request->response: one call in, one answer back, and it
// blocks until that answer comes. That's perfect for a single call, but it means
// "do N things at once" can't happen on the wire alone — the calls stack, and one
// slow call (an iOS session that takes 2 minutes to provision) holds up the rest.
//
// The broker fixes that with one idea you already know from Kafka/Redis, sized for
// one machine: a DEQUE — work goes in one end, results come out the other, and a
// pool of workers in the middle runs them all at once.
//
//		efferent-in  ──►  [ job job job ]  ──►  afferent-out
//		(POST /submit)      workers run          (GET /events, pushed as each finishes)
//		                    them concurrently
//
//	  - efferent  = carried OUT (the commands/creates we send into the world)
//	  - afferent  = carried BACK (the results the world sends us)
//
// Three endpoints, nothing more:
//
//	POST /submit  — hand it a list of jobs (each is a wire-shaped HTTP call).
//	                Returns immediately; the work runs in the background.
//	GET  /events  — subscribe. Each finished job is pushed to you as it lands,
//	                in completion order (fast iOS-vs-Android no longer serialize).
//	GET  /health  — last time we heard anything, how many subscribers, the counts.
//
// A "job" carries an authProfile name, not a secret — the broker resolves the key
// from the environment (LT_USERNAME / LT_ACCESS_KEY), so the caller never holds it.
// Each job runs with its OWN timeout, so a slow create can't stall a fast one.
//
// stdlib only. net/http for the server and the calls, channels for the deque.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Job is one wire-shaped call to run. Same four fields as the wire, plus an
// optional auth profile (resolved here, below the trust boundary) and a per-job
// timeout so one slow call can't hold up the others.
type Job struct {
	ID          string            `json:"id"`
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        string            `json:"body,omitempty"`
	AuthProfile string            `json:"authProfile,omitempty"`
	TimeoutSec  int               `json:"timeoutSec,omitempty"`
}

// Result is what comes back out the afferent end — the job's outcome, stamped
// with how long it took and when it finished.
type Result struct {
	ID        string `json:"id"`
	Status    int    `json:"status"`
	Body      string `json:"body,omitempty"`
	Err       string `json:"err,omitempty"`
	ElapsedMs int64  `json:"elapsedMs"`
	At        string `json:"at"`
}

// broker holds the live state: who's listening (subscribers), the running totals,
// and when we last did anything (so silence becomes a readable signal).
type broker struct {
	mu          sync.Mutex
	subscribers map[int]chan Result
	nextSub     int

	queued    atomic.Int64
	running   atomic.Int64
	done      atomic.Int64
	lastHeard atomic.Int64 // unix nanos
}

func newBroker() *broker {
	return &broker{subscribers: map[int]chan Result{}}
}

// subscribe registers a consumer and returns its channel + an unsubscribe func.
// The channel is buffered so one slow consumer can't block the fan-out.
func (b *broker) subscribe() (int, chan Result, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.nextSub
	b.nextSub++
	ch := make(chan Result, 64)
	b.subscribers[id] = ch
	return id, ch, func() {
		b.mu.Lock()
		delete(b.subscribers, id)
		close(ch)
		b.mu.Unlock()
	}
}

// publish fans a result out to every subscriber. Non-blocking: if a consumer's
// buffer is full it's dropped for that consumer (v1 — "log the gap, see if they
// notice"), never stalling the others.
func (b *broker) publish(r Result) {
	b.lastHeard.Store(time.Now().UnixNano())
	b.mu.Lock()
	for _, ch := range b.subscribers {
		select {
		case ch <- r:
		default:
		}
	}
	b.mu.Unlock()
}

// resolveAuth turns a profile name into a Basic credential from the environment.
// The secret stays here — the caller only ever names the profile. (v1: one
// account, from LT_USERNAME / LT_ACCESS_KEY.)
func resolveAuth(profile string) (string, bool) {
	if profile == "" {
		return "", false
	}
	u, k := os.Getenv("LT_USERNAME"), os.Getenv("LT_ACCESS_KEY")
	if u == "" || k == "" {
		return "", false
	}
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(u+":"+k)), true
}

// run executes one job and returns its Result. Each call gets its own client
// timeout, so a 2-minute iOS create and a 60-second Android create live their
// own lives instead of one blocking the other.
func run(j Job) Result {
	start := time.Now()
	stamp := func(status int, body, errStr string) Result {
		return Result{
			ID: j.ID, Status: status, Body: body, Err: errStr,
			ElapsedMs: time.Since(start).Milliseconds(),
			At:        time.Now().Format(time.RFC3339),
		}
	}

	to := j.TimeoutSec
	if to <= 0 {
		to = 300 // generous by default — cloud real-device creates are slow
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(to)*time.Second)
	defer cancel()

	var bodyReader io.Reader
	if j.Body != "" {
		bodyReader = strings.NewReader(j.Body)
	}
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(j.Method), j.URL, bodyReader)
	if err != nil {
		return stamp(0, "", "build request: "+err.Error())
	}
	for k, v := range j.Headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" && j.Body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if h, ok := resolveAuth(j.AuthProfile); ok {
		req.Header.Set("Authorization", h)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return stamp(0, "", err.Error())
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return stamp(resp.StatusCode, string(raw), "")
}

func main() {
	addr := flag.String("addr", ":7700", "address to listen on")
	flag.Parse()
	b := newBroker()
	b.lastHeard.Store(time.Now().UnixNano())

	// POST /submit — the efferent-in end. Accept N jobs, run each in its own
	// goroutine (the worker pool), and return at once. Results arrive later on
	// /events as each job finishes.
	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var jobs []Job
		if err := json.NewDecoder(r.Body).Decode(&jobs); err != nil {
			http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
			return
		}
		b.lastHeard.Store(time.Now().UnixNano())
		for _, j := range jobs {
			b.queued.Add(1)
			go func(j Job) {
				b.queued.Add(-1)
				b.running.Add(1)
				res := run(j)
				b.running.Add(-1)
				b.done.Add(1)
				b.publish(res) // afferent-out: push the result to every consumer
			}(j)
		}
		writeJSON(w, map[string]any{"accepted": len(jobs)})
	})

	// GET /events — the afferent-out end. Subscribe (Server-Sent Events) and get
	// each finished job pushed to you live, in completion order.
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		_, ch, unsub := b.subscribe()
		defer unsub()
		fmt.Fprintf(w, ": subscribed\n\n")
		flusher.Flush()

		for {
			select {
			case <-r.Context().Done():
				return
			case res := <-ch:
				data, _ := json.Marshal(res)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	})

	// GET /health — silence made readable: when we last heard anything, who's
	// listening, and the running totals.
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		subs := len(b.subscribers)
		b.mu.Unlock()
		last := time.Unix(0, b.lastHeard.Load())
		writeJSON(w, map[string]any{
			"subscribers":     subs,
			"queued":          b.queued.Load(),
			"running":         b.running.Load(),
			"done":            b.done.Load(),
			"lastHeard":       last.Format(time.RFC3339),
			"silentForSec":    int64(time.Since(last).Seconds()),
			"authProfileSeen": os.Getenv("LT_USERNAME") != "",
		})
	})

	log.Printf("broker up on %s — POST /submit, GET /events, GET /health", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
