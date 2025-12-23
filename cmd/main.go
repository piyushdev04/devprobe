package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"devprobe/probe"
	"devprobe/tui"
)

func main() {
	// flags
	tuiEnabled := flag.Bool("tui", false, "use interactive TUI")
	timeout := flag.Duration("timeout", 5*time.Second, "request timeout (e.g. 5s)")
	retries := flag.Int("retries", 1, "number of retries on failure")
	concurrency := flag.Int("c", 1, "number of concurrent requests")
	requests := flag.Int("n", 1, "total number of requests")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: devprobe <url> [-tui] [-c concurrency] [-n requests]")
		os.Exit(1)
	}

	url := flag.Arg(0)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// TUI MODE
	if *tuiEnabled {
		p := tea.NewProgram(
			tui.New(ctx, url, *retries, *concurrency, *requests),
			tea.WithAltScreen(),
		)

		if err := p.Start(); err != nil {
			fmt.Println("TUI error:", err)
			os.Exit(1)
		}
		return
	}

	// CLI MODE
	runCLI(ctx, url, *retries, *concurrency, *requests)
}

func runCLI(
	ctx context.Context,
	url string,
	retries, concurrency, requests int,
) {
	results := make(chan probe.Result)
	var wg sync.WaitGroup

	wg.Add(4)
	go probe.DNS(ctx, url, retries, results, &wg)
	go probe.TCP(ctx, url, retries, results, &wg)
	go probe.TLS(ctx, url, retries, results, &wg)
	go probe.HTTP(ctx, url, retries, results, &wg)

	go func() {
		wg.Wait()
		close(results)
	}()

	fmt.Println("🔍 Probing:", url)

	var collected []probe.Result
	for r := range results {
		collected = append(collected, r)
	}

	sort.Slice(collected, func(i, j int) bool {
		return collected[i].Order < collected[j].Order
	})

	for _, r := range collected {
		fmt.Println(r.Format())
	}

	if requests > 1 {
		printLoad(ctx, url, concurrency, requests, retries)
	}
}

func printLoad(
	ctx context.Context,
	url string,
	concurrency, requests, retries int,
) {
	fmt.Println("\n⚡ Load Test")

	stats := probe.Load(ctx, url, concurrency, requests, retries)

	sort.Slice(stats.Latencies, func(i, j int) bool {
		return stats.Latencies[i] < stats.Latencies[j]
	})

	p95 := stats.Latencies[int(float64(len(stats.Latencies))*0.95)]

	fmt.Printf("Requests: %d\n", stats.Total)
	fmt.Printf("Concurrency: %d\n", concurrency)
	fmt.Printf("Success: %d\n", stats.Success)
	fmt.Printf("Errors: %d\n", stats.Errors)
	fmt.Printf("Avg latency: %dms\n", avg(stats.Latencies))
	fmt.Printf("P95 latency: %dms\n", p95)
}

func avg(nums []int64) int64 {
	var sum int64
	for _, n := range nums {
		sum += n
	}
	return sum / int64(len(nums))
}
