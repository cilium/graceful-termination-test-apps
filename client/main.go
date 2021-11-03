package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"time"
)

// Following constants should be synced with cilium CI.

const MSG_SIZE = 256 // Needs to be synced with server
const IO_TIME_OUT = 5 * time.Second
const SERVER_CLOSE_MSG = "server shutdown"

func Run(servAddr *net.TCPAddr) {
	var (
		conn           *net.TCPConn
		err            error
	)

	for i := 0; i < 10; i++ {
		conn, err = net.DialTCP("tcp", nil, servAddr)
		if err == nil {
			break
		}
		fmt.Printf("connect to %s failed : %s. Re-connecting\n", servAddr, err)
		time.Sleep(1 * time.Second)
	}
	panicOnErr("dial tcp failed", err)
	defer conn.Close()

	request := make([]byte, MSG_SIZE)
	_, err = rand.Read(request)
	panicOnErr("rand.Read", err)
	for {
		reply := make([]byte, MSG_SIZE)

		err = conn.SetWriteDeadline(time.Now().Add(IO_TIME_OUT))
		panicOnErr("setWriteDeadline", err)
		_, err = conn.Write(request)
		panicOnErr("write failed", err)

		err = conn.SetReadDeadline(time.Now().Add(IO_TIME_OUT))
		panicOnErr("setReadDeadline", err)
		n, err := conn.Read(reply)
		panicOnErr("read failed", err)

		fmt.Println("client received reply")

		if bytes.Compare(request, reply[:n]) != 0 {
			if string(reply[:n]) == SERVER_CLOSE_MSG {
				fmt.Println(string(reply[:n]))
				os.Exit(0)
			} else {
				panic(fmt.Sprintf("invalid server reply(%v) != request(%v)", reply, request))
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func panicOnErr(ctx string, err error) {
	if err != nil {
		panic(fmt.Sprintf("%s: %s", ctx, err))
	}
}

func main() {
	var (
		err      error
		servAddr *net.TCPAddr
	)
	remote := os.Args[1]

	for i := 0; i < 10; i++ {
		if servAddr, err = net.ResolveTCPAddr("tcp", remote); err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	panicOnErr(fmt.Sprintf("resolve tcp address failed [%s]:", remote), err)

	for {
		Run(servAddr)
	}
}
