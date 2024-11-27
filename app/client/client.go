package client

import (
	"log/slog"
	"net"
)

func client() {
	conn, err := net.Dial("tcp", "localhost:6379")
	if err != nil {
		slog.Error("Error connecting to server: ", "err", err.Error());
		return
	}
	defer conn.Close()

	// sending a resp message in the form of bulk string
	message := "$5\r\nHello\r\n"
	if _, err := conn.Write([]byte(message)); err != nil {
		slog.Error("Error writing to server: ", "err", err.Error())
		return
	}

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		slog.Error("Error reading from the server: ", "err", err.Error())
		return
	}

	slog.Info("Server response: ", "serverResponse", string(buffer[:n]))
}