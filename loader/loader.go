package loader

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pilosa/tools"
)

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)
	var config = flags.String("config", "", "specify a toml config file")

	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	cfg := &Config{}
	if *config != "" {
		fmt.Fprintf(stdout, "loading config from %s\n", *config)
		if err := cfg.Load(*config); err != nil {
			return fmt.Errorf("error loading config %q: %w", *config, err)
		}
	}

	var command string
	commands := flags.Args()
	if len(commands) == 1 {
		command = commands[0]
	}

	l := &loader{
		stderr: stderr,
		stdout: stdout,
		stats:  newStats(),
	}

	l.Println(`Version: ` + tools.Version + `+  Build Time: ` + tools.BuildTime + "\n")

	switch command {
	case "config":
		return l.printConfig(cfg)
	default:
		return l.load(cfg)
	}
}

type loader struct {
	stderr io.Writer
	stdout io.Writer
	stats  *stats
}

func (l *loader) Println(args ...interface{}) {
	fmt.Fprintln(l.stderr, args...)
}
func (l *loader) Printf(f string, args ...interface{}) {
	fmt.Fprintf(l.stderr, f, args...)
}

func (l *loader) printConfig(cfg *Config) error {
	if len(cfg.Tasks) == 0 {
		l.Println("no config provided, printing a sample config\n")
		cfg = DemoConfig()
	}
	return cfg.Print(l.stdout)
}

func (l *loader) load(cfg *Config) error {
	if len(cfg.Tasks) == 0 {
		return errors.New("No tasks found in config")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create a channel to block everyone on so that all tasks are spun up and ready to go before loading
	start := make(chan struct{})

	// create a wg to coordinate all tasks
	var wg = &sync.WaitGroup{}
	wg.Add(len(cfg.Tasks))

	for _, t := range cfg.Tasks {
		go l.launchTask(ctx, t, start, wg)
	}
	// let all tasks start
	close(start)

	// start monitor loop
	go l.printStats(ctx)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	// Block until one of the signals above is received
	<-signalCh
	l.Println("Signal received, initializing clean shutdown...")
	cancel()

	finished := make(chan struct{})
	go func(finished chan struct{}) {
		wg.Wait()
		close(finished)
	}(finished)

	// Block again until another signal is received, a shutdown timeout elapses,
	// or the Command is gracefully closed
	l.Println("Waiting for clean shutdown...")
	select {
	case <-signalCh:
		return errors.New("second signal received, initializing hard shutdown")
	case <-time.After(time.Second * 30):
		return errors.New("time limit reached, initializing hard shutdown")
	case <-finished:
		l.Println("shutdown completed")
		return nil
	}
}

func (l *loader) launchTask(ctx context.Context, t Task, start chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	// calculate time offset between connections
	offset := time.Duration(int64(time.Duration(t.Delay)) / int64(t.Connections))

	var cwg = &sync.WaitGroup{}
	cwg.Add(t.Connections)
	for i := 0; i < t.Connections; i++ {
		go l.childTask(ctx, cwg, start, t, offset*time.Duration(i))
	}

	<-ctx.Done()

	cwg.Wait()
}

func (l *loader) childTask(ctx context.Context, wg *sync.WaitGroup, start chan struct{}, t Task, offset time.Duration) {
	defer wg.Done()

	client := http.Client{}

	req, _ := http.NewRequest("POST", t.URL, bytes.NewBufferString(t.Query))

	proc := func() {
		_, err := client.Do(req)
		if err != nil {
			l.stats.error(t.Query)
		}
		l.stats.success(t.Query)
	}

	// wait to start
	<-start
	// don't start all children at the same time, try to keep them spaced apart
	time.Sleep(offset)
	proc()

	// begin interval loop
	tick := time.NewTicker(time.Duration(t.Delay))
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			proc()
		case <-ctx.Done():
			return
		}
	}
}

func (l *loader) printStats(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := l.stats.counts()
			for k, v := range stats {
				l.Printf("%s\t success: %d\t errors: %d\n", k, v[0], v[1])
			}
		}
	}
}
