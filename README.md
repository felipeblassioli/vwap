# VWAP

This project hosts a command-line application that calculates VWAP (Volume-weighted average price) in realtime from 
[Coinbase's Websocket feed API](https://docs.cloud.coinbase.com/exchange/docs/websocket-overview), specifically for the
`matches` channel.

To use the application you may:
1. Use a [prebuilt binary](https://github.com/felipeblassioli/vwap/releases) for your platform.
2. Build and run locally.

The command-line requires no parameters, but optionally you may specify them:

```
Usage:
  -addr string
        Coinbase's websocket feed URI (default "wss://ws-feed.exchange.coinbase.com")
  -products string
        Comma separated list of coinbase's product IDs (default "BTC-USD,ETH-USD,ETH-BTC")
  -window int
        The width of the window for calculating VWAP values (default 200)
```

## Build and run locally

### Build from source

For building locally Go version >= 1.18 is required.

```bash
$ go build -o vwap cmd/vwap/main.go
$ ./vwap
# Output:
# 2022/09/16 02:48:16 ETH-BTC:  0.0745800000000000
# 2022/09/16 02:48:16 ETH-USD:  1475.0199999999999818
# 2022/09/16 02:48:16 BTC-USD:  19775.1399999999994179
# 2022/09/16 02:48:16 BTC-USD:  19775.1399999999994179
# 2022/09/16 02:48:16 BTC-USD:  19775.1399999999994179
# 2022/09/16 02:48:16 BTC-USD:  19775.1399999999994179
# 2022/09/16 02:48:16 BTC-USD:  19775.1399999999994179
```

## Design and assumptions

### Project structure

```
cmd	
  vwap	
pkg	
  coinbase  Package coinbase provides a client that interacts with Coinbase's Websocket Feed.
    wstest    Package wstest provides utilities for Websocket testing.
  ringbuf   Package ringbuf provides a ring buffer data structure.
  vwap	    Package vwap provides a Volume-weighted average price calculator.
```

The command-line application is in the `cmd/vwap` directory

Libraries are located inside the `pkg` directory and 
each one has an `go.doc` file describing their purpose and what they provide.

### Code overview

In a nutshell, the  relies on a few concepts: ring buffer, pipelines and cancelletion.

**Ring buffer**

The [ring buffer](https://en.wikipedia.org/wiki/Circular_buffer) was used to buffer
the data-stream sent by Coinbase's websocket server and the buffered data-stream
was used to calculate the VWAP within a sliding window.

The `ringbuf` package implementation uses a mutex. But it is possible to do a lockless
implementation which may be more performant. 

See implementations of lockless ring buffers:

  - Lockless Ring Buffer Design:
    https://www.kernel.org/doc/Documentation/trace/ring-buffer-design.txt
  - A channel based ring buffer in go:
    https://tanzu.vmware.com/content/blog/a-channel-based-ring-buffer-in-go

**Pipelines and cancellation**

As described in [Go Concurrency Patterns: Pipelines and cancellation](https://go.dev/blog/pipelines):

> What is a pipeline?
> 
> There’s no formal definition of a pipeline in Go; it’s just one of many kinds 
> of concurrent programs. Informally, a pipeline is a series of stages connected 
> by channels, where each stage is a group of goroutines running the same function. 
> In each stage, the goroutines:
>   - receive values from upstream via inbound channels
>   - perform some function on that data, usually producing new values
>   - send values downstream via outbound channels
> Each stage has any number of inbound and outbound channels, except the first 
> and last stages, which have only outbound or inbound channels, respectively. 
> The first stage is sometimes called the source or producer; the last stage, t
> he sink or consumer.

Using the above terminology, the command-line application can be seen as the following pipeline:

1. `MatchesWatcher goroutine`: 
   1. Reads [Match](https://docs.cloud.coinbase.com/exchange/docs/websocket-channels#match) 
   data from [coinbase Websocket feed](https://docs.cloud.coinbase.com/exchange/docs/websocket-channels#match)
   via websocket.
   2. Sends the data downstream via go channel.
2. `VWAPCalculator goroutine`:
   1. Receives Match data from upstream
   2. Calculates a new VWAP value
   3. Sends the up-to-date VWAP value (within the sliding window) downstream
3. `Printer goroutine`:
   1. Receives VWAP values from upstream
   2. Outputs these values to STDOUT

For coordinating the goroutines it was used the standard library `errgroup` package and all goroutines and sub-goroutines
cooperate by respecting the `context` package cancellation signal.
