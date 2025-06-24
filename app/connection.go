package main

import (
	"bufio"
	"fmt"
	"net"
)

// Connection handles a single client connection
type Connection struct {
	conn   net.Conn
	parser *RESPParser
	writer *RESPWriter
}

// NewConnection creates a new connection handler
func NewConnection(conn net.Conn) *Connection {
	reader := bufio.NewReader(conn)
	bufWriter := bufio.NewWriter(conn)

	return &Connection{
		conn:   conn,
		parser: NewRESPParser(reader),
		writer: NewRESPWriter(bufWriter),
	}
}

// Handle processes incoming commands from the client
func (c *Connection) Handle(server *RedisServer) {
	defer c.conn.Close()

	for {
		// Parse incoming RESP message
		value, err := c.parser.Parse()
		if err != nil {
			fmt.Printf("Error parsing RESP: %v\n", err)
			return
		}

		// Convert RESP array to command arguments
		if value.Type != Array {
			c.writer.WriteError("expected array")
			continue
		}

		args := c.extractArgs(value)
		if args == nil {
			continue // Error already sent
		}

		// Handle the command
		err = server.HandleCommand(args, c.writer)
		if err != nil {
			fmt.Printf("Error handling command: %v\n", err)
		}
	}
}

// extractArgs extracts string arguments from a RESP array
func (c *Connection) extractArgs(value RESPValue) []string {
	args := make([]string, len(value.Array))

	for i, arg := range value.Array {
		switch arg.Type {
		case BulkString:
			args[i] = arg.Bulk
		case SimpleString:
			args[i] = arg.Str
		default:
			c.writer.WriteError("invalid argument type")
			return nil
		}
	}

	return args
}
