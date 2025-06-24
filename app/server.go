package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// KeyValue represents a stored value with optional expiry
type KeyValue struct {
	Value     string
	ExpiresAt *time.Time
}

// CommandHandler interface for handling Redis commands
type CommandHandler interface {
	Handle(args []string, writer *RESPWriter) error
}

// PingHandler handles PING commands
type PingHandler struct{}

func (h *PingHandler) Handle(args []string, writer *RESPWriter) error {
	return writer.WriteSimpleString("PONG")
}

// EchoHandler handles ECHO commands
type EchoHandler struct{}

func (h *EchoHandler) Handle(args []string, writer *RESPWriter) error {
	if len(args) < 2 {
		return writer.WriteError("wrong number of arguments for 'echo' command")
	}
	return writer.WriteBulkString(args[1])
}

// SetHandler handles SET commands
type SetHandler struct {
	server *RedisServer
}

func (h *SetHandler) Handle(args []string, writer *RESPWriter) error {
	if len(args) < 3 {
		return writer.WriteError("wrong number of arguments for 'set' command")
	}

	key := args[1]
	value := args[2]
	var expiresAt *time.Time

	// Parse EX/PX options
	for i := 3; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return writer.WriteError("syntax error")
		}

		option := strings.ToUpper(args[i])
		switch option {
		case "EX":
			seconds, err := strconv.Atoi(args[i+1])
			if err != nil || seconds <= 0 {
				return writer.WriteError("value is not an integer or out of range")
			}
			expiry := time.Now().Add(time.Duration(seconds) * time.Second)
			expiresAt = &expiry
		case "PX":
			milliseconds, err := strconv.Atoi(args[i+1])
			if err != nil || milliseconds <= 0 {
				return writer.WriteError("value is not an integer or out of range")
			}
			expiry := time.Now().Add(time.Duration(milliseconds) * time.Millisecond)
			expiresAt = &expiry
		default:
			return writer.WriteError("syntax error")
		}
	}

	// Thread-safe write to data store
	h.server.mutex.Lock()
	h.server.data[key] = KeyValue{
		Value:     value,
		ExpiresAt: expiresAt,
	}
	h.server.mutex.Unlock()

	return writer.WriteSimpleString("OK")
}

// GetHandler handles GET commands
type GetHandler struct {
	server *RedisServer
}

func (h *GetHandler) Handle(args []string, writer *RESPWriter) error {
	if len(args) != 2 {
		return writer.WriteError("wrong number of arguments for 'get' command")
	}

	key := args[1]

	// Thread-safe read from data store
	h.server.mutex.Lock()
	h.server.cleanupExpired(key) // Clean expired key first
	kv, exists := h.server.data[key]
	h.server.mutex.Unlock()

	if !exists {
		// Return null bulk string for non-existent key
		return writer.WriteNullBulkString()
	}

	return writer.WriteBulkString(kv.Value)
}

// TTLHandler handles TTL commands
type TTLHandler struct {
	server *RedisServer
}

func (h *TTLHandler) Handle(args []string, writer *RESPWriter) error {
	if len(args) != 2 {
		return writer.WriteError("wrong number of arguments for 'ttl' command")
	}

	key := args[1]

	h.server.mutex.Lock()
	h.server.cleanupExpired(key) // Clean expired key first
	kv, exists := h.server.data[key]
	h.server.mutex.Unlock()

	if !exists {
		return writer.WriteInteger(-2) // key doesn't exist
	}

	if kv.ExpiresAt == nil {
		return writer.WriteInteger(-1) // no expiry
	}

	remaining := time.Until(*kv.ExpiresAt)
	if remaining <= 0 {
		// Key expired, clean it up
		h.server.mutex.Lock()
		delete(h.server.data, key)
		h.server.mutex.Unlock()
		return writer.WriteInteger(-2)
	}

	return writer.WriteInteger(int(remaining.Seconds()))
}

// RedisServer represents the Redis server
type RedisServer struct {
	handlers map[string]CommandHandler
	data     map[string]KeyValue
	mutex    sync.RWMutex
}

// NewRedisServer creates a new Redis server
func NewRedisServer() *RedisServer {
	server := &RedisServer{
		handlers: make(map[string]CommandHandler),
		data:     make(map[string]KeyValue),
	}

	// Register command handlers
	server.handlers["PING"] = &PingHandler{}
	server.handlers["ECHO"] = &EchoHandler{}
	server.handlers["SET"] = &SetHandler{server: server}
	server.handlers["GET"] = &GetHandler{server: server}
	server.handlers["TTL"] = &TTLHandler{server: server}

	return server
}

// isExpired checks if a key has expired
func (s *RedisServer) isExpired(key string) bool {
	kv, exists := s.data[key]
	if !exists {
		return false
	}
	if kv.ExpiresAt == nil {
		return false // no expiry
	}
	return time.Now().After(*kv.ExpiresAt)
}

// cleanupExpired removes an expired key
func (s *RedisServer) cleanupExpired(key string) {
	if s.isExpired(key) {
		delete(s.data, key)
	}
}

// HandleCommand processes a Redis command
func (s *RedisServer) HandleCommand(cmd []string, writer *RESPWriter) error {
	if len(cmd) == 0 {
		return writer.WriteError("empty command")
	}

	command := strings.ToUpper(cmd[0])
	handler, exists := s.handlers[command]
	if !exists {
		return writer.WriteError(fmt.Sprintf("unknown command '%s'", command))
	}

	return handler.Handle(cmd, writer)
}
