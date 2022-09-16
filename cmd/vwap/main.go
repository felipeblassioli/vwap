package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/felipeblassioli/vwap/pkg/coinbase"
	"github.com/felipeblassioli/vwap/pkg/vwap"
)

const debug = false // enable for debugging

// NewSigKillContext returns a Context that cancels when os.Interrupt
// or os.Kill is received
func NewSigKillContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	//nolint:gomnd // Channel size is 2 because we could be notified twice
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()

	return ctx
}

// runMatchesWatcher connects to Coinbase's websocket feed and subscribes
// to the "matches" channel for a given product (example: BTC-USD).
//
// It outputs all received messages (Feed updates) to the matches go channel.
func runMatchesWatcher(
	ctx context.Context,
	matches chan<- coinbase.Match,
	addr string,
	productID string,
	windowWidth int,
) error {
	c := coinbase.NewClient()
	err := c.Connect(ctx, addr)
	if err != nil {
		return err
	}

	subscription, err := c.Subscribe(ctx, productID, windowWidth)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			<-subscription.Done()
			return ctx.Err()
		case match := <-subscription.C:
			matches <- match
		}
	}
}

// runVWAPCalculator receives coinbase's Matches feed updates via `updates`
// channel parameter, calculates the VWAP and send the result to the printer.
func runVWAPCalculator(
	ctx context.Context,
	updates <-chan coinbase.Match,
	printer chan<- string,
	windowWidth int,
	name string,
) error {
	calc := vwap.NewCalculator(windowWidth)
	for {
		select {
		case <-ctx.Done():
			if debug {
				//nolint:forbidigo // Removed by the compiler
				log.Println("Stopping vwap.Calculator updates", name)
			}
			return ctx.Err()
		case m := <-updates:
			//nolint:gocritic // Shadowing the package in this scope is ok for clarity
			vwap, err := calc.Update(m.Price, m.Size)
			if err != nil {
				return err
			}
			printer <- vwap
		}
	}
}

func runPrinter(ctx context.Context, printer chan string, product string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case s := <-printer:
			//nolint:forbidigo // Printer purpose is printing to stdout
			log.Println(product+": ", s)
		}
	}
}

func main() {
	var (
		addr = flag.String(
			"addr",
			"wss://ws-feed.exchange.coinbase.com",
			"Coinbase's websocket feed URI",
		)
		products = flag.String(
			"products",
			"BTC-USD,ETH-USD,ETH-BTC",
			"Comma separated list of coinbase's product IDs",
		)
		windowWidth = flag.Int(
			"window",
			//nolint:gomnd // Default value is an educated guess
			200,
			"The width of the window for calculating VWAP values",
		)
	)
	flag.Parse()

	g, ctx := errgroup.WithContext(NewSigKillContext())

	// As per coinbase's documentation best practices:
	// Spread subscriptions over more than one websocket client connection.
	//
	// See: https://docs.cloud.coinbase.com/exchange/docs/websocket-best-practices
	for _, p := range strings.Split(*products, ",") {
		// Pipeline for each product p:
		// chan []coinbase.Match -> chan string -> os.Stdout
		// matches               -> vwap        -> printer
		func(p string) {
			matches := make(chan coinbase.Match, *windowWidth)
			printer := make(chan string, *windowWidth)

			g.Go(func() error {
				return runPrinter(ctx, printer, p)
			})

			g.Go(func() error {
				defer close(printer)
				return runVWAPCalculator(
					ctx,
					matches,
					printer,
					*windowWidth,
					p,
				)
			})

			g.Go(func() error {
				defer close(matches)
				return runMatchesWatcher(
					ctx,
					matches,
					*addr,
					p,
					*windowWidth,
				)
			})
		}(p)
	}

	// TODO: recover from panics or let it fail? Cleanup will not be reached if it panics
	<-ctx.Done()
	if err := g.Wait(); err != nil && err != context.Canceled {
		//nolint:forbidigo // Printing error to os.Stderr on program exit is intentional
		log.Fatal(err)
	}
}
