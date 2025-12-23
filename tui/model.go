package tui

import (
	"context"
	"fmt"
	"sync"
	"sort"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"devprobe/probe"
)

var (
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F5F"))
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	boxStyle   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2)
)

type resultMsg probe.Result
type loadMsg probe.LoadStats
type errMsg error

type probeResultsMsg struct {
	results []probe.Result
	ctx     context.Context
	url     string
	retries int
	c       int
	n       int
}

type Model struct {
	ctx         context.Context
	url         string
	retries     int
	concurrency int
	requests    int

	results []probe.Result
	load    *probe.LoadStats

	avgLatency int64
	p95Latency int64

	err     error
	done    bool
}

func New(
	ctx context.Context,
	url string,
	retries int,
	concurrency int,
	requests int,
) Model {
	return Model{
		ctx:         ctx,
		url:         url,
		retries:     retries,
		concurrency: concurrency,
		requests:    requests,
	}
}

func (m Model) Init() tea.Cmd {
	return runProbes(m.ctx, m.url, m.retries, m.concurrency, m.requests)
}

func runProbes(
	ctx context.Context,
	url string,
	retries int,
	concurrency int,
	requests int,
) tea.Cmd {
	return func() tea.Msg {
		resultsCh := make(chan probe.Result)
		var wg sync.WaitGroup

		wg.Add(4)
		go probe.DNS(ctx, url, retries, resultsCh, &wg)
		go probe.TCP(ctx, url, retries, resultsCh, &wg)
		go probe.TLS(ctx, url, retries, resultsCh, &wg)
		go probe.HTTP(ctx, url, retries, resultsCh, &wg)

		go func() {
			wg.Wait()
			close(resultsCh)
		}()

		var results []probe.Result
		for r := range resultsCh {
			results = append(results, r)
		}

		return probeResultsMsg{
			results: results,
			ctx:     ctx,
			url:     url,
			retries: retries,
			c:       concurrency,
			n:       requests,
		}
	}
}

func runLoad(
	ctx context.Context,
	url string,
	concurrency int,
	requests int,
	retries int,
) tea.Cmd {
	return func() tea.Msg {
		stats := probe.Load(ctx, url, concurrency, requests, retries)
		return loadMsg(stats)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case probeResultsMsg:
		m.results = msg.results

		if msg.n > 1 {
			return m, runLoad(
				msg.ctx,
				msg.url,
				msg.c,
				msg.n,
				msg.retries,
			)
		}

		m.done = true
		return m, nil

	case loadMsg:
	stats := probe.LoadStats(msg)
	m.load = &stats

	if len(stats.Latencies) > 0 {
			sort.Slice(stats.Latencies, func(i, j int) bool {
				return stats.Latencies[i] < stats.Latencies[j]
			})

			var sum int64
			for _, l := range stats.Latencies {
				sum += l
			}

			m.avgLatency = sum / int64(len(stats.Latencies))
			m.p95Latency = stats.Latencies[int(float64(len(stats.Latencies))*0.95)]
		}

		m.done = true
		return m, nil


	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) View() string {
	title := labelStyle.Render("devprobe")

	body := fmt.Sprintf("🔍 Probing %s\n\n", m.url)

	if len(m.results) == 0 {
		body += "⏳ Running probes...\n"
	} else {
		for _, r := range m.results {
			status := okStyle.Render("✔")
			if r.Err != nil {
				status = errStyle.Render("✖")
			}
			body += fmt.Sprintf(
				"%-15s %s %dms %s\n",
				r.Name,
				status,
				r.Duration,
				r.Extra,
			)
		}
	}

	if m.load != nil {
		body += "\n⚡ Load Test\n"
		body += fmt.Sprintf("Requests: %d\n", m.load.Total)
		body += fmt.Sprintf("Concurrency: %d\n", m.concurrency)
		body += fmt.Sprintf("Success: %d\n", m.load.Success)
		body += fmt.Sprintf("Errors: %d\n", m.load.Errors)
		body += fmt.Sprintf("Avg latency: %dms\n", m.avgLatency)
		body += fmt.Sprintf("P95 latency: %dms\n", m.p95Latency)
	}

	body += "\nPress q to quit"

	return boxStyle.Render(title + "\n\n" + body)
}