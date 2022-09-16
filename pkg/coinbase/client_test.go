package coinbase

import (
	"context"
	"encoding/json"
	"io"
	"math/rand"
	"net"
	"os"
	"testing"

	ws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"

	"github.com/felipeblassioli/vwap/pkg/coinbase/wstest"
)

func Test_Connect(t *testing.T) {
	var (
		err error
		ctx = context.Background()
		c   = NewClient()
	)

	// err.DNSError: Invalid URI
	t.Run("Invalid URI", func(t *testing.T) {
		var e *net.DNSError
		err = c.Connect(ctx, "ws://INVALID.sandbox.exchange.coinbase.com")
		assert.ErrorAs(t, err, &e)
	})

	t.Run("Success", func(t *testing.T) {
		// 1. Arrange
		var (
			s = wstest.NewFakeCoinbaseServer()
			c = NewClient()
		)

		// 3. Act
		err = c.Connect(context.Background(), s.URL)

		// 3. Assert
		assert.Nil(t, err)
	})
}

func Test_SubscribeAndUnsubscribe(t *testing.T) {
	// Subscribing to an unexisting product results in an error:
	// {"type":"error","message":"Failed to subscribe","reason":"BTC-XXX is not a valid product"}
	t.Run("Invalid productID", func(t *testing.T) {
		// 1. Arrange
		var (
			ctx              = context.Background()
			s                = wstest.NewFakeCoinbaseServer()
			c                = NewClient()
			respondSubscribe = func() {
				data := []byte("{\"type\":\"error\",\"message\":\"Failed to subscribe\",\"reason\":\"BTC-XXX is not a valid product\"}")
				s.WriteMessage(wstest.Message{Type: ws.TextMessage, Data: data})
			}
		)
		c.Connect(ctx, s.URL)

		// 2. Act
		s.SetReadMessageHandler(func(c *ws.Conn, messageType int, message []byte) {
			respondSubscribe()
			s.SetReadMessageHandler(nil)
		})
		_, err := c.Subscribe(ctx, "BTC-XXX", 0)

		// 3. Assert
		assert.Error(t, err)
	})

	t.Run("Successful subscription", func(t *testing.T) {
		// 1. Arrange
		// Fixtures: matches expected to be received by the subscription
		var exp []Match
		{
			f, _ := os.Open("testdata/matches-1.json")
			defer f.Close()

			b, _ := io.ReadAll(f)
			json.Unmarshal(b, &exp)
		}
		var (
			ctx       = context.Background()
			c         = NewClient()
			s         = wstest.NewFakeCoinbaseServer()
			productID = "BTC-USD"
			// Server response to Subscribe message
			respondSubscribe = func() {
				resp, _ := json.Marshal(Subscriptions{
					Type: "subscriptions",
					Channels: []MessageChannel{
						{
							Name:       "matches",
							ProductIds: []string{productID},
						},
					},
				})

				s.WriteMessage(wstest.Message{Type: ws.TextMessage, Data: resp})
			}
			// Server will send the matches via websocket
			streamMatchesData = func(matches []Match) {
				for _, match := range matches {
					// Fake server currently uses raw bytes (UTF-8 encoded text)
					b, _ := json.Marshal(match)
					s.WriteMessage(wstest.Message{Type: ws.TextMessage, Data: b})
				}
			}
		)
		c.Connect(ctx, s.URL)

		// 2. Act
		// Simulate response to Subscribe message
		s.SetReadMessageHandler(func(*ws.Conn, int, []byte) {
			respondSubscribe()
			s.SetReadMessageHandler(nil)
		})

		// Subscribes to Websocket Feed updates
		// windowWidth is not relevant for this test
		ww := rand.Intn(len(exp)) + 1
		subscription, _ := c.Subscribe(ctx, productID, ww)

		// Server streams Websocket Feed updates
		go streamMatchesData(exp)

		// Record all received updates
		var act []Match
		for i := 0; i < len(exp); i++ {
			match := <-subscription.C
			act = append(act, match)
		}

		// 3. Assert
		assert.Equal(t, exp, act)
	})
}
