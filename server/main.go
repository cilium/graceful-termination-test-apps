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
const IO_TIME_OUT = 5 * time.Second
const SERVER_CLOSE_MSG = "server shutdown"
const NUM_WORKERS = 5
const GRACEFUL_TERMINATION_PERIOD = 10 * time.Second

type tcpServer struct {
	shutdown chan struct{}
	activeConnWg sync.WaitGroup
}

func (s *tcpServer) handleSignals()  {
	var sigs = make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGTERM)
	if sg := <- sigs; sg == syscall.SIGTERM {
		fmt.Println("graceful shutdown on SIGTERM")
		close(s.shutdown)
	}
}

func (s *tcpServer) serve(listener *net.TCPListener)  {
	conn, err := listener.AcceptTCP()
	if errors.Is(err, net.ErrClosed) {
		// server shutting down.
		return
	} else {
		panicOnErr("accept failed", err)
	}
	fmt.Printf("received connection from %v\n", conn.RemoteAddr())
	s.activeConnWg.Add(1)
	defer s.activeConnWg.Done()
	buf := make([]byte, MSG_SIZE)
	for  {
		select {
		case <-s.shutdown:
			err = conn.SetWriteDeadline(time.Now().Add(IO_TIME_OUT))
			panicOnErr("write deadline failed", err)
			_, err = conn.Write([]byte(SERVER_CLOSE_MSG))
			panicOnErr("write failed", err)
			time.Sleep(GRACEFUL_TERMINATION_PERIOD)
			conn.Close()
			return
		default:
			_, err := conn.Read(buf)
			panicOnErr("read failed", err)
			err = conn.SetWriteDeadline(time.Now().Add(IO_TIME_OUT))
			panicOnErr("write deadline failed", err)
			_, err = conn.Write(buf)
			panicOnErr("write failed", err)
		}
	}
}

func panicOnErr(ctx string, err error) {
	if err != nil {
		panic(fmt.Sprintf("%s: %s", ctx, err))
	}
}

func main()  {
	port := os.Args[1]

	addr, err := net.ResolveTCPAddr("tcp", ":" + port)
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
			fmt.Println("exiting")
			return
		default:
		}
	}
}

