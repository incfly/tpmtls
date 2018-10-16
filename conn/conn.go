// binary conn is to play with file descriptor limit when you don't close connection...
// Command to show the
// $
// To update teh
// $
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

var (
	iter      = flag.Int("iter", 1000, "help message for flagname")
	closeConn = flag.Bool("close_conn", true, "whether to clean the connection")
)

func main() {
	flag.Parse()
	// Listen on TCP port 2000 on all available unicast and
	// anycast IP addresses of the local system.
	l, err := net.Listen("tcp", ":2000")

	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	// Start server
	go func() {
		for {
			// Wait for a connection.
			conn, err := l.Accept()
			if err != nil {
				log.Fatal(err)
			}
			// Handle the connection in a new goroutine.
			// The loop then returns to accepting, so that
			// multiple connections may be served concurrently.
			go func(c net.Conn) {
				// Echo all incoming data.
				io.Copy(c, c)
				if *closeConn {
					// Shut down the connection.
					c.Close()
				}
			}(conn)
		}
	}()
	// Start client part.
	for i := 0; i < *iter; i++ {
		fmt.Println("Start client ", i)
		conn, err := net.Dial("tcp", ":2000")
		if err != nil {
			fmt.Printf("failed to dial %v\n", err)
			os.Exit(-1)
		}
		fmt.Println("dial OK")
		if _, err := conn.Write([]byte("hi")); err != nil {
			fmt.Println("Write:", err)
			return
		}
		resp := make([]byte, 1024)
		_, err = conn.Read(resp)
		if err != nil {
			fmt.Println("Read:", err)
			return
		}
		if *closeConn {
			conn.Close()
		}
	}
	fmt.Println("iter value ", *iter)
}
