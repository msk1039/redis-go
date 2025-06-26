# redis-go

A simple Redis server clone written in Go. This project implements a subset of the Redis protocol and core commands, allowing you to interact with it using any Redis client.

## Features

- RESP protocol parsing and serialization
- Handles multiple client connections concurrently
- Implements core Redis commands:
  - `PING`
  - `ECHO <message>`
  - `SET <key> <value> [EX seconds|PX milliseconds]`
  - `GET <key>`
  - `TTL <key>`
- Supports key expiry with `EX` (seconds) and `PX` (milliseconds) options
- Thread-safe in-memory key-value store

## Getting Started

### Prerequisites
- [Go](https://golang.org/dl/) 1.18 or newer

### Clone the repository
```sh
git clone https://github.com/your-username/redis-go.git
cd redis-go/app
```

### Build and Run
```sh
go build -o redis-server main.go connection.go resp.go server.go
./redis-server
```

The server will start on port `6379` by default.

### Usage
You can connect to your server using the official `redis-cli` or any Redis client:

```sh
redis-cli -p 6379
```

Try out commands like:
```
PING
ECHO "Hello, World!"
SET mykey myvalue
GET mykey
SET tempkey tempval EX 10
TTL tempkey
```

## License
MIT
