package coinbase

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	ws "github.com/gorilla/websocket"
)

const debug = false // enable for debugging

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 5 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	//nolint:gomnd // the values are arbitrary
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 32768
)

// MatchesClient connects via websocket to Coinbase's Websocket Feed to stream
// Match updates from the "matches" channel
type MatchesClient struct {
	conn *ws.Conn
}

func NewClient() *MatchesClient {
	return &MatchesClient{}
}

// Connect connects to Coinbase's Websocket feed
func (c *MatchesClient) Connect(ctx context.Context, addr string) error {
	//nolint:bodyclose // body is close for debugging, which is removed by the compiler if debug == false
	conn, resp, err := ws.DefaultDialer.DialContext(
		ctx,
		addr,
		// As per coinbase's documentation best practices:
		// Includes the `Sec-WebSocket-Extensions: permessage-deflate` header
		// to allow for compression, which will lower bandwidth consumption
		// with minimal impact to CPU / memory.
		http.Header{
			"Sec-WebSocket-Extensions": {"permessage-deflate"},
		},
	)

	if debug && resp != nil {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			//nolint:forbidigo // Removed by compiler
			log.Fatalln(err)
		}

		//nolint:forbidigo // Removed by compiler
		log.Println("ws.Dial", addr, "http.response:", len(b), string(b))
	}

	if err != nil {
		return fmt.Errorf("connect error: %w", err)
	}

	// Configures connection for disconnect detection
	c.conn = conn

	c.conn.SetCloseHandler(func(code int, text string) error {
		if debug {
			log.Println("closeHandler: ", code, text) //nolint:forbidigo // Removed by compiler
		}

		// As per rfc6455 section 5.5.1:
		// If an endpoint receives a Close frame and did not previously send a
		// Close frame, the endpoint MUST send a Close frame in response. (When
		// sending a Close frame in response, the endpoint typically echos the
		// status code it received.)  It SHOULD do so as soon as practical.
		//
		// See: https://www.rfc-editor.org/rfc/rfc6455#section-5.5.1
		message := ws.FormatCloseMessage(code, "")

		return c.conn.WriteControl(ws.CloseMessage, message, time.Now().Add(writeWait))
	})

	return nil
}

// Subscribe subscribes to "matches" channel from Coinbase Websocket Feed.
// It puts all FeedUpdate in the channel Subscription.C
func (c *MatchesClient) Subscribe(
	_ context.Context,
	productID string,
	windowWidth int,
) (*Subscription, error) {
	subscribe := Subscribe{
		Type: "subscribe",
		Channels: []MessageChannel{
			{
				Name:       "matches",
				ProductIds: []string{productID},
			},
		},
	}

	if err := c.conn.WriteJSON(subscribe); err != nil {
		return nil, fmt.Errorf("subscribe error: %w", err)
	}

	// TODO: should we expose this message to the consumer?
	message := Message{}
	err := c.conn.ReadJSON(&message)
	if err != nil {
		return nil, fmt.Errorf("subscribe failed: %w", err)
	}
	if debug {
		//nolint:forbidigo // Removed by compiler
		log.Println("subscribe response: ", message)
	}
	if message.Type == "error" {
		return nil, fmt.Errorf("subscribe error: %s", message.Reason)
	}

	s := NewSubscription(c.conn, windowWidth)

	go s.matchWatcher()

	return s, nil
}

// Unsubscribe unsubscribes from "matches" channel for a single product.
// A zero value for productID means unsubscribing from "matches" channel
// entirely.
func (c *MatchesClient) Unsubscribe(productID string) error {
	var productIDs []string
	if productID != "" {
		productIDs = append(productIDs, productID)
	}
	unsubscribe := &Unsubscribe{
		Type:       "unsubscribe",
		ProductIds: productIDs,
		Channels:   []string{"matches"},
	}

	//nolint:errcheck // Gorilla *ws.Conn implementation always returns nil
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.conn.WriteJSON(unsubscribe); err != nil {
		return fmt.Errorf("unsubscribe error: %w", err)
	}

	return nil
}
