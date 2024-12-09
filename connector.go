package simconnect

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"syscall"
	"time"

	"github.com/bmurray/simconnect-go/client"
	"github.com/cenkalti/backoff/v4"
)

// Receiver is the interface for receiving data from SimConnect
type Receiver interface {
	// Start is called when the receiver is started
	// it gets called after the connection is established
	// and whenever a reconnection happens
	// the context is cancelled when the connection is lost
	// this may be called multiple times if the connection is lost and re-established
	Start(ctx context.Context, sc *client.SimConnect)

	// Update is called whenever a new data packet is received
	// the context is cancelled when the connection is lost
	// this may be called multiple times over the life of the connection
	Update(ctx context.Context, sc *client.SimConnect, ppData *client.RecvSimobjectDataByType)
}

// Connector is the main struct for connecting to SimConnect
type Connector struct {
	// simconnect *simconnect.SimConnect
	name      string
	receivers []Receiver
	cycle     time.Duration

	log *slog.Logger
}

// ConnectorOption is a function that sets options on the Connector
type ConnectorOption func(*Connector)

// WithReceiver adds a receiver to the connector
// you can add multiple receivers
func WithReceiver(r Receiver) ConnectorOption {
	return func(c *Connector) {
		c.receivers = append(c.receivers, r)
	}
}

// WithCycle sets the cycle time for the connector
// the connector will dispatch data every cycle
func WithCycle(cycle time.Duration) ConnectorOption {
	return func(c *Connector) {
		c.cycle = cycle
	}
}

// WithLogger sets the logger for the connector
func WithLogger(l *slog.Logger) ConnectorOption {
	return func(c *Connector) {
		c.log = l.With("module", "simconnect")
	}
}

// NewConnector creates a new connector
// you can pass options to the connector
func NewConnector(name string, opts ...ConnectorOption) *Connector {
	c := &Connector{
		name:  name,
		cycle: 100 * time.Millisecond,
		log:   slog.Default().With("name", name, "module", "simconnect"),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Start starts the connector
// it will connect to SimConnect and start the receivers
// this is BLOCKING, and will terminate at the first disconnect
func (c *Connector) Start(ctx context.Context) {
	if err := c.connect(ctx); err != nil {
		c.log.Error("Connection Terminated Abnormally", "err", err)
		return
	}
}

// StartReconnect starts the connector with reconnect
// it will connect to SimConnect and start the receivers
// this is BLOCKING, and will reconnect on disconnect
// This is a simple wrapper around Start that adds a exponential backoff
func (c *Connector) StartReconnect(ctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0
	for {
		t := time.Now()
		select {
		case <-ctx.Done():
			return
		default:
			c.Start(ctx)
		}
		d := time.Since(t)
		if d > 90*time.Second {
			bo.Reset()
		}
		nxt := bo.NextBackOff()
		if nxt == backoff.Stop {
			c.log.Debug("Reconnect stopped")
			return
		}
		c.log.Info("Restarting Connection", "run_duration", d, "next", nxt)
		select {
		case <-ctx.Done():
			return
		case <-time.After(nxt):
			c.log.Debug("Reconnect")
		}
	}
}

func (c *Connector) connect(ctx context.Context) error {
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()

	sc, err := client.New(c.name)
	if err != nil && errors.Is(err, syscall.Errno(0)) {
		return fmt.Errorf("cannot connect to SimConnect: %w", err)
	} else if err != nil {
		return fmt.Errorf("cannot connect to SimConnect: %w", err)
	}
	defer func() {
		if err := sc.Close(); err != nil {
			c.log.Error("Cannot close SimConnect", "error", err)
		}
	}()

	for _, r := range c.receivers {
		r.Start(ctx2, sc)
	}
	dispatcher := time.NewTicker(c.cycle)
	defer dispatcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-dispatcher.C:
			// Dispatch
			err := dispatchFn(ctx2, sc, func(x *client.RecvSimobjectDataByType) error {
				for _, r := range c.receivers {
					r.Update(ctx2, sc, x)
				}
				return nil
			})
			if err != nil {
				if errors.Is(err, ErrGetNextDispatch) {
					return fmt.Errorf("cannot dispatch: %w", err)
				} else if !errors.Is(err, syscall.Errno(0)) {
					c.log.Warn("Dispatch error, not critical", "error", err)
				}
			}
		}
	}
}

// ConnectorError is the error type for the connector
type ConnectorError string

func (e ConnectorError) Error() string { return string(e) }

const (
	// ErrE_FAIL is the error for E_FAIL
	ErrE_FAIL ConnectorError = "E_FAIL"
	// ErrGetNextDispatch is the error for GetNextDispatch
	ErrGetNextDispatch ConnectorError = "GetNextDispatch"
)

func dispatchFn(ctx context.Context, s *client.SimConnect, fn func(*client.RecvSimobjectDataByType) error) error {
	ppData, r1, err := s.GetNextDispatch()
	if r1 < 0 {
		if uint32(r1) == client.E_FAIL {
			return fmt.Errorf("GetNextDispatch error E_FAIL: %d %w %T", r1, err, err)
		} else {
			return fmt.Errorf("GetNextDispatch error: %d %w", r1, ErrGetNextDispatch)
		}
	}
	recvInfo := *(*client.Recv)(ppData)
	switch recvInfo.ID {
	case client.RECV_ID_EXCEPTION:
		recvErr := *(*client.RecvException)(ppData)
		err = client.RecvException(recvErr)
		return fmt.Errorf("SIMCONNECT_RECV_ID_EXCEPTION: %w", err)
	case client.RECV_ID_OPEN:
		recvOpen := *(*client.RecvOpen)(ppData)
		err = client.RecvOpen(recvOpen)
		// Ignore open message
		// return fmt.Errorf("SIMCONNECT_RECV_ID_OPEN %w", err)
		return nil
	case client.RECV_ID_EVENT:
		recvEvent := *(*client.RecvEvent)(ppData)
		err = client.RecvEventError(recvEvent)
		return fmt.Errorf("SIMCONNECT_RECV_ID_EVENT %w", err)
	case client.RECV_ID_SIMOBJECT_DATA_BYTYPE:
		x := (*client.RecvSimobjectDataByType)(ppData)
		return fn(x)
	default:
		return fmt.Errorf("recvInfo.dwID unknown: %d", recvInfo.ID)
	}
}
