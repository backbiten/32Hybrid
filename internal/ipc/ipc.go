// Package ipc provides the inter-process communication primitives used for
// message passing between 32Hybrid userland services and the Legacy32
// nano-kernel subsystems.
//
// Messages are typed envelopes with a string payload. Channels connect a
// sender and a receiver; the in-process MemChannel is used for unit testing
// and single-binary integration scenarios. A network-backed channel can be
// substituted for multi-process or cross-host deployment without changing
// the calling code.
//
// Example – sending a request and receiving the response:
//
//	ch := ipc.NewMemChannel(8)
//	ch.Send(ipc.Message{
//	    ID:      "req-1",
//	    Type:    ipc.TypeRequest,
//	    From:    "userland-app",
//	    To:      "network",
//	    Payload: `{"action":"start"}`,
//	})
//	msg, err := ch.Recv(ctx)
package ipc

import (
	"context"
	"fmt"
	"sync"
)

// MessageType classifies the intent of an IPC message.
type MessageType int

const (
	// TypeRequest is a call from a client to a service expecting a reply.
	TypeRequest MessageType = iota
	// TypeResponse is a reply sent by a service to a previous TypeRequest.
	TypeResponse
	// TypeEvent is a one-way, fire-and-forget notification.
	TypeEvent
	// TypeError carries an error condition back to the original requester.
	TypeError
)

// String returns the human-readable name of the message type.
func (t MessageType) String() string {
	switch t {
	case TypeRequest:
		return "request"
	case TypeResponse:
		return "response"
	case TypeEvent:
		return "event"
	case TypeError:
		return "error"
	default:
		return fmt.Sprintf("unknown(%d)", int(t))
	}
}

// Message is the envelope exchanged over an IPC Channel.
// All fields are plain strings to keep the protocol language-agnostic;
// callers may encode structured data (e.g. JSON) in Payload.
type Message struct {
	// ID is an opaque correlation token set by the sender.
	// Responses should echo the ID of the corresponding request.
	ID string

	// Type classifies the message (request, response, event, error).
	Type MessageType

	// From is the logical name of the sending service or process.
	From string

	// To is the logical name of the intended receiving service or process.
	To string

	// Payload carries the message body, typically JSON-encoded.
	Payload string
}

// Channel is the interface for bidirectional IPC message passing.
// Implementations must be safe for concurrent use.
type Channel interface {
	// Send places msg onto the channel. Returns an error if the channel is
	// full (blocking callers should loop or use a context deadline).
	Send(msg Message) error

	// Recv blocks until a message is available or ctx is cancelled.
	// Returns context.Canceled or context.DeadlineExceeded on timeout.
	Recv(ctx context.Context) (Message, error)

	// Close signals that no further messages will be sent.  Pending Recv
	// calls will drain buffered messages and then return an error.
	Close() error
}

// MemChannel is a buffered, in-process Channel backed by a Go channel.
// It is intended for unit testing and same-process integration scenarios.
type MemChannel struct {
	mu     sync.Mutex
	ch     chan Message
	closed bool
}

// NewMemChannel creates a MemChannel with the given buffer size.
// A bufSize of 0 produces an unbuffered (synchronous) channel.
func NewMemChannel(bufSize int) *MemChannel {
	return &MemChannel{ch: make(chan Message, bufSize)}
}

// Send implements Channel.
func (c *MemChannel) Send(msg Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("ipc: channel is closed")
	}
	select {
	case c.ch <- msg:
		return nil
	default:
		return fmt.Errorf("ipc: channel buffer is full")
	}
}

// Recv implements Channel. It drains any buffered messages before
// returning an error when the channel has been closed.
func (c *MemChannel) Recv(ctx context.Context) (Message, error) {
	select {
	case <-ctx.Done():
		return Message{}, ctx.Err()
	case msg, ok := <-c.ch:
		if !ok {
			return Message{}, fmt.Errorf("ipc: channel is closed")
		}
		return msg, nil
	}
}

// Close implements Channel. After Close returns, any further Send calls
// return an error. Recv continues to drain buffered messages and then
// returns an error once the buffer is empty.
func (c *MemChannel) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("ipc: channel already closed")
	}
	c.closed = true
	close(c.ch)
	return nil
}
