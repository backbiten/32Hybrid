package ipc_test

import (
	"context"
	"testing"
	"time"

	"github.com/backbiten/32Hybrid/internal/ipc"
)

func TestMemChannel_SendRecv(t *testing.T) {
	ch := ipc.NewMemChannel(4)

	want := ipc.Message{
		ID:      "req-1",
		Type:    ipc.TypeRequest,
		From:    "userland-app",
		To:      "network",
		Payload: `{"action":"start"}`,
	}
	if err := ch.Send(want); err != nil {
		t.Fatalf("Send: %v", err)
	}

	ctx := context.Background()
	got, err := ch.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("ID: got %q, want %q", got.ID, want.ID)
	}
	if got.Type != want.Type {
		t.Errorf("Type: got %v, want %v", got.Type, want.Type)
	}
	if got.Payload != want.Payload {
		t.Errorf("Payload: got %q, want %q", got.Payload, want.Payload)
	}
}

func TestMemChannel_FullBuffer(t *testing.T) {
	ch := ipc.NewMemChannel(1)
	if err := ch.Send(ipc.Message{ID: "first"}); err != nil {
		t.Fatalf("Send first: %v", err)
	}
	// Buffer is full; second send should fail immediately.
	if err := ch.Send(ipc.Message{ID: "second"}); err == nil {
		t.Error("Send on full buffer: expected error, got nil")
	}
}

func TestMemChannel_RecvContextCancel(t *testing.T) {
	ch := ipc.NewMemChannel(4)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := ch.Recv(ctx)
	if err == nil {
		t.Error("Recv with cancelled context: expected error, got nil")
	}
}

func TestMemChannel_ClosePreventsNewSend(t *testing.T) {
	ch := ipc.NewMemChannel(4)
	if err := ch.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := ch.Send(ipc.Message{ID: "late"}); err == nil {
		t.Error("Send after Close: expected error, got nil")
	}
}

func TestMemChannel_CloseTwice(t *testing.T) {
	ch := ipc.NewMemChannel(4)
	_ = ch.Close()
	if err := ch.Close(); err == nil {
		t.Error("second Close: expected error, got nil")
	}
}

func TestMemChannel_MultipleMessages(t *testing.T) {
	ch := ipc.NewMemChannel(8)
	ids := []string{"a", "b", "c"}
	for _, id := range ids {
		if err := ch.Send(ipc.Message{ID: id, Type: ipc.TypeEvent}); err != nil {
			t.Fatalf("Send %q: %v", id, err)
		}
	}
	ctx := context.Background()
	for _, want := range ids {
		got, err := ch.Recv(ctx)
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		if got.ID != want {
			t.Errorf("ID: got %q, want %q", got.ID, want)
		}
	}
}

func TestMessageType_String(t *testing.T) {
	cases := []struct {
		mt   ipc.MessageType
		want string
	}{
		{ipc.TypeRequest, "request"},
		{ipc.TypeResponse, "response"},
		{ipc.TypeEvent, "event"},
		{ipc.TypeError, "error"},
		{ipc.MessageType(99), "unknown(99)"},
	}
	for _, tc := range cases {
		if got := tc.mt.String(); got != tc.want {
			t.Errorf("MessageType(%d).String() = %q, want %q", int(tc.mt), got, tc.want)
		}
	}
}
