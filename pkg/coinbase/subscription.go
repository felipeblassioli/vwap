package coinbase

import (
	"context"
	"fmt"
	"log"
	"time"

	ws "github.com/gorilla/websocket"
)

// Match messages represents a trade that occurred between two orders.
//
// The aggressor or taker order is the one executing immediately after being
// received  and the maker order is a resting order on the book.
type Match struct {
	// Type is "last_match" for the first message received after subscribing
	// or "match" for all others.
	Type    string `json:"type"`
	TradeID int    `json:"trade_id"`
	// Sequence numbers are increasing integer values for each product, with
	// each new message being exactly one sequence number greater than the
	// one before it.
	//
	// Sequence numbers that are greater than one integer value from the
	// previous number indicate that a message has been dropped.
	//
	// Sequence numbers that are less than the previous number can be
	// ignored or represent a message that has arrived out of order.
	Sequence     int       `json:"sequence"`
	MakerOrderID string    `json:"maker_order_id"`
	TakerOrderID string    `json:"taker_order_id"`
	Time         time.Time `json:"time"`
	ProductID    string    `json:"product_id"`
	Size         string    `json:"size"`
	Price        string    `json:"price"`
	// Side indicates the maker order side.
	// If the side is sell this indicates the maker was a sell order and
	// the match is considered an up-tick.
	// A buy side match is a down-tick.
	Side string `json:"side"`
}

// Subscription watches Coinbase Websocket Feed Match updates.
// All updates are sent to the go channel C.
type Subscription struct {
	conn *ws.Conn
	done chan error
	C    chan Match
}

func NewSubscription(conn *ws.Conn, windowWidth int) *Subscription {
	return &Subscription{
		conn: conn,
		//nolint:gomnd // 2 is maximum number of statements that send to this channel
		done: make(chan error, 2),
		C:    make(chan Match, windowWidth),
	}
}

// matchWatcher watches Coinbase Websocket Feed updates.
// All feed updates are put in the go channel Subscription.C
func (s *Subscription) matchWatcher() {
	var reterr error
	conn := s.conn

	// As per Coinbase's best practices documentation:
	// Connected clients should increase their web socket receive buffer to
	// the largest configurable amount possible (given any client library or
	// infrastructure limitations), due to the potential volume of data for
	// any given product.
	//
	// For Gorilla's websocket it will return ws.ErrReadLimit if the message
	// exceeds this maximum size
	conn.SetReadLimit(maxMessageSize)

	// start sending Ping messages to peer
	go func() {
		// If pinger stops then we will violate the ReadDeadline for
		// this watcher.
		s.done <- pinger(context.Background(), s.conn)
	}()

	// Set read deadline to a time less than next expected pong.
	if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		s.done <- err
		return
	}
	conn.SetPongHandler(func(string) error {
		if debug {
			//nolint:forbidigo // Removed by compiler
			log.Println("pong")
		}
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		match := Match{}

		// Applications must break out of the application's read loop when this method
		// returns a non-nil error value. Errors returned from this method are
		// permanent. Once this method returns a non-nil error, all subsequent calls to
		// this method return the same error.
		//
		// That happens because calls are all made over a single tcp connection, so
		// once the ReadDeadline is reached on the connection, all further Reads will fail.
		// There's no way for the client to selectively read the responses (which may come
		// back out of order) when the connection is essentially closed.
		err := conn.ReadJSON(&match)

		if err != nil {
			err = fmt.Errorf("matchWatcher read failed: %w", err)
			reterr = err
			break
		}

		// FIXME: if the channel is full (because of slow consume) this may block forever
		// coinbase will disconnect us after 5 seconds according to their documentation
		s.C <- match
	}
	s.done <- reterr
}

// pinger sends periodically Ping control messages to the peer.
//
// See rfc6455 for more about Ping messages:
//   - https://www.rfc-editor.org/rfc/rfc6455#section-5.5.2
func pinger(ctx context.Context, conn *ws.Conn) error {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			//nolint:errcheck // Gorilla *ws.Conn implementation always returns nil
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := conn.WriteMessage(ws.PingMessage, []byte{})
			if err != nil {
				return err
			}
			if debug {
				//nolint:forbidigo // Removed by compiler
				log.Println("ping")
			}
		}
	}
}

// Stop stops the subscriptions from reading feed updates.
// It also closes the chanel s.C
func (s *Subscription) Stop() {
	<-s.done
	close(s.C)
	close(s.done)
}

// Done returns a channel that blocks until the subscription is stopped.
func (s *Subscription) Done() <-chan error {
	return s.done
}
