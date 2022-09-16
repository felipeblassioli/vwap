package wstest

import (
	"bytes"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	ws "github.com/gorilla/websocket"
)

const debug = false // enable for debugging

type ReadMessageHandler func(c *ws.Conn, messageType int, message []byte)

type Message struct {
	// Type is either ws.TextMessage or ws.BinaryMessage
	Type int
	// Data is the message content
	Data []byte
}

// FakeCoinbaseServer fakes Coinbase's Websocket Feed API.
type FakeCoinbaseServer struct {
	*httptest.Server

	// recordedMessages slice has all the read messages by the server
	recordedMessages    []Message
	readMessageHandler  ReadMessageHandler
	messagesToBeWritten chan Message
	handleConnState     func(net.Conn, http.ConnState)
}

// NewFakeCoinbaseServer starts and returns a new FakeCoinbaseServer.
//
// The caller should call Close when finished, to shut it down.
func NewFakeCoinbaseServer() *FakeCoinbaseServer {
	var (
		s = &FakeCoinbaseServer{
			messagesToBeWritten: make(chan Message, 1),
		}
		upgrader = ws.Upgrader{}
	)

	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()

		go func() {
			for {
				msg := <-s.messagesToBeWritten
				err = c.WriteMessage(msg.Type, msg.Data)
				if err != nil {
					break
				}
			}
		}()

		for {
			if err := c.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
				panic(err)
			}
			mt, data, err := c.ReadMessage()
			if err != nil {
				panic(err)
			}
			s.recordedMessages = append(s.recordedMessages, Message{mt, data})
			if debug {
				//nolint:forbidigo // Removed by the compiler
				log.Printf("recorded Message{%d %s}\n", mt, string(data))
			}
			if s.readMessageHandler != nil {
				s.readMessageHandler(c, mt, data)
			}
		}
	}))
	s.URL = "ws" + strings.TrimPrefix(s.URL, "http")

	s.Server.Config.ConnState = func(conn net.Conn, state http.ConnState) {
		if s.handleConnState != nil {
			s.handleConnState(conn, state)
		}
	}
	return s
}

// SetConnectionStateHandler registers a function that is called when a client connection
// changes state
func (s *FakeCoinbaseServer) SetConnectionStateHandler(h func(net.Conn, http.ConnState)) {
	s.handleConnState = h
}

// SetReadMessageHandler is called for every Data read by the FakeServer
func (s *FakeCoinbaseServer) SetReadMessageHandler(h ReadMessageHandler) {
	s.readMessageHandler = h
}

// ReceivedMessage returns true if any of the recorded messages is equal
// to the target message.
func (s *FakeCoinbaseServer) ReceivedMessage(target Message) bool {
	for _, m := range s.recordedMessages {
		if target.Type == m.Type && bytes.Equal(target.Data, m.Data) {
			return true
		}
	}
	return false
}

func (s *FakeCoinbaseServer) WriteMessage(msg Message) {
	s.messagesToBeWritten <- msg
}
