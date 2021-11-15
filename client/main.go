package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

// Following constants should be synced with cilium CI.

const MSG_SIZE = 256 // Needs to be synced with server
const RECEIVED_SERVER_MSG = "client received reply"

func Run(servAddr *net.TCPAddr) {
	var (
		conn *net.TCPConn
		err  error
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
	fmt.Printf("connected to %v \n", conn.RemoteAddr())
	defer conn.Close()

	request := make([]byte, MSG_SIZE)
	_, err = rand.Read(request)
	panicOnErr("rand.Read", err)
	for {
		reply := make([]byte, MSG_SIZE)

		_, err = conn.Write(request)
		if err != nil && err == io.EOF {
			fmt.Printf("server closed the connection %v\n", err)
			conn.Close()
			os.Exit(0)
		}
		panicOnErr("write failed", err)

		n, err := conn.Read(reply)
		if err != nil && err == io.EOF {
			fmt.Printf("server closed the connection %v\n", err)
			conn.Close()
			os.Exit(0)
		}
		panicOnErr("read failed", err)

		fmt.Println(RECEIVED_SERVER_MSG)

		if bytes.Compare(request, reply[:n]) != 0 {
			panic(fmt.Sprintf("invalid server reply(%v) != request(%v)", reply, request))
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

	for i := 0; i < 60; i++ {
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
