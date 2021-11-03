package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Following constants should be synced with cilium CI.

const MSG_SIZE = 256 // Needs to be synced with client
const IO_TIME_OUT = 1 * time.Second
const NUM_WORKERS = 5
const GRACEFUL_TERMINATION_PERIOD = 15 * time.Second
const RECEIVED_CLIENT_CONN = "received connection from"
const TERMINATION_MSG = "terminating on SIGTERM"

type tcpServer struct {
	shutdown     chan struct{}
	activeConnWg sync.WaitGroup
}

func (s *tcpServer) handleSignals() {
	var sigs = make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGTERM)
	if sg := <-sigs; sg == syscall.SIGTERM {
		fmt.Println(TERMINATION_MSG)
		close(s.shutdown)
	}
}

func (s *tcpServer) serve(listener *net.TCPListener) {
	conn, err := listener.AcceptTCP()
	if errors.Is(err, net.ErrClosed) {
		// server shutting down.
		return
	} else {
		panicOnErr("accept failed", err)
	}
	fmt.Printf("%s "+"%v\n", RECEIVED_CLIENT_CONN, conn.RemoteAddr())
	s.activeConnWg.Add(1)
	defer s.activeConnWg.Done()
	buf := make([]byte, MSG_SIZE)
	for {
		select {
		case <-s.shutdown:
			conn.Close()
			return
		default:
			_, _ = conn.Read(buf)
			_ = conn.SetWriteDeadline(time.Now().Add(IO_TIME_OUT))
			_, err = conn.Write(buf)
		}
	}
}

func panicOnErr(ctx string, err error) {
	if err != nil {
		panic(fmt.Sprintf("%s: %s", ctx, err))
	}
}

func main() {
	port := os.Args[1]

	addr, err := net.ResolveTCPAddr("tcp", ":"+port)
	panicOnErr(fmt.Sprintf("resolve tcp address %s failed", addr), err)

	listener, err := net.ListenTCP("tcp", addr)
	panicOnErr("listen failed", err)

	server := tcpServer{
		shutdown: make(chan struct{}),
	}
	go server.handleSignals()
	// Start bounded number of handlers to request incoming requests.
	for i := 0; i < NUM_WORKERS; i++ {
		go server.serve(listener)
	}
	for {
		select {
		case <-server.shutdown:
			// Unblock AcceptTCP (idle) goroutines
			listener.Close()
			// Wait until active connections are drained
			server.activeConnWg.Wait()
			time.Sleep(GRACEFUL_TERMINATION_PERIOD)
			fmt.Println("exiting")
			return
		default:
		}
	}
}
