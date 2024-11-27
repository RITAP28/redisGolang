package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"redisGolang/redisproto"
	// "strings"
)

var _ = net.Listen
var _ = os.Exit

const (
	receiveBufferSize = 1024
)

func main() {
	fmt.Println("Logs from program will appear here")

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		slog.Error("Failed to bind to port 6379")
		os.Exit(1)
	}

	defer func() {
		l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			slog.Error("Error accepting connection: ", "err", err.Error())
			os.Exit(1)
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	serverResponse := []byte("+PONG\r\n")

	// a buffer of size 1024 bytes is created to store incoming messages from the client
	// as data from the tcp connection is read in chunks
	// and a buffer is used to store each chunk temporarily
	buf := make([]byte, receiveBufferSize)

	for {
		datalength, err := conn.Read(buf)
		if err != nil {
			// EOF means End Of File, means the client has closed connections
			if !errors.Is(err, io.EOF) {
				slog.Error("reading", "err", err)
			}

			slog.Error("Error reading from connection: ", "err", err.Error())
			return
		}

		if datalength == 0 {
			fmt.Println("No data available to read")
			return
		}

		if _, err := conn.Write(serverResponse); err != nil {
			slog.Error("Error writing to connection: ", "err", err.Error())
		}

		// mainly for Efficient Buffer Management, optimizing memory usage
		// reads the meaningful data sent from the client by slicing the buffer
		// for example the buffer is of fixed size 1024 and suppose the client sent only 5 bytes of data
		// so, cmd will store those 5 bytes and not those extra empty bytes
		cmd := buf[:datalength]

		slog.Info("Received", "len", len(cmd), "str", cmd)

		// Another method of reading and writing to the data received from the client
		// messages := strings.Split(string(buf), "\r\n")
		// for _, message := range messages {
		// 	switch message {
		// 	case "PING":
		// 		if _, err := conn.Write(serverResponse); err != nil {
		// 			slog.Error("Error writing to connection: ", "err", err.Error())
		// 			return
		// 		}
		// 	default:
		// 		fmt.Println("Received data: ", string(buf))
		// 	}
		// }

		_, resp := redisproto.ReadNextRESP(cmd)

		if resp.Type == redisproto.BulkString {
			fmt.Println("Received bulk string: ", string(resp.Data))
			response := fmt.Sprintf("+%s\r\n", resp.Data)
			conn.Write([]byte(response))
		} else {
			conn.Write([]byte("-ERR unsupported command\r\n"))
		}

	}
}
