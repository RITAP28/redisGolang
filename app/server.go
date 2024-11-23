package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

var _ = net.Listen
var _ = os.Exit

func main() {
	fmt.Println("Logs from program will appear here");

	l, err := net.Listen("tcp", "0.0.0.0:6379");
	if err != nil {
		fmt.Println("Failed to bind to port 6379");
		os.Exit(1);
	}

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error());
		os.Exit(1);
	}
	
	// a buffer of size 1024 bytes is created to store incoming messages from the client
	// as data from the tcp connection is read in chunks
	// and a buffer is used to store each chunk temporarily
	buf := make([]byte, 1024);

	for {
		dataLength, err := conn.Read(buf);
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("Connection closed")
				break
			}
			fmt.Println("Error reading from connection: ", err.Error());
			break
		}

		if dataLength == 0 {
			fmt.Println("No data read");
			break
		}

		messages := strings.Split(string(buf), "\r\n")

		for _, message := range messages {
			switch message {
			case "PING":
				conn.Write([]byte("+PONG\r\n"));
			default:
				fmt.Println("Received Data: ", string(buf))
			}
		}
	}
}