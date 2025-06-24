package main

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// RESPType represents the type of RESP data
type RESPType byte

const (
	SimpleString RESPType = '+'
	Error        RESPType = '-'
	Integer      RESPType = ':'
	BulkString   RESPType = '$'
	Array        RESPType = '*'
)

// RESPValue represents a RESP protocol value
type RESPValue struct {
	Type  RESPType
	Str   string
	Num   int
	Bulk  string
	Array []RESPValue
}

// RESPParser handles parsing RESP protocol messages
type RESPParser struct {
	reader *bufio.Reader
}

// NewRESPParser creates a new RESP parser
func NewRESPParser(reader *bufio.Reader) *RESPParser {
	return &RESPParser{reader: reader}
}

// Parse reads and parses a RESP value from the connection
func (p *RESPParser) Parse() (RESPValue, error) {
	for {
		typeByte, err := p.reader.ReadByte()
		if err != nil {
			return RESPValue{}, err
		}

		// Skip stray \r or \n characters
		if typeByte == '\r' || typeByte == '\n' {
			continue
		}

		switch RESPType(typeByte) {
		case Array:
			return p.parseArray()
		case BulkString:
			return p.parseBulkString()
		case SimpleString:
			return p.parseSimpleString()
		case Error:
			return p.parseError()
		case Integer:
			return p.parseInteger()
		default:
			return RESPValue{}, fmt.Errorf("unknown RESP type: %c", typeByte)
		}
	}
}

// parseArray parses a RESP array
func (p *RESPParser) parseArray() (RESPValue, error) {
	line, err := p.readLine()
	if err != nil {
		return RESPValue{}, err
	}

	count, err := strconv.Atoi(line)
	if err != nil {
		return RESPValue{}, err
	}

	array := make([]RESPValue, count)
	for i := 0; i < count; i++ {
		val, err := p.Parse()
		if err != nil {
			return RESPValue{}, err
		}
		array[i] = val
	}

	return RESPValue{Type: Array, Array: array}, nil
}

// parseBulkString parses a RESP bulk string
func (p *RESPParser) parseBulkString() (RESPValue, error) {
	line, err := p.readLine()
	if err != nil {
		return RESPValue{}, err
	}

	length, err := strconv.Atoi(line)
	if err != nil {
		return RESPValue{}, err
	}

	if length == -1 {
		return RESPValue{Type: BulkString, Bulk: ""}, nil
	}

	bulk := make([]byte, length)
	_, err = p.reader.Read(bulk)
	if err != nil {
		return RESPValue{}, err
	}

	// Read the trailing \r\n
	p.reader.ReadByte() // \r
	p.reader.ReadByte() // \n

	return RESPValue{Type: BulkString, Bulk: string(bulk)}, nil
}

// parseSimpleString parses a RESP simple string
func (p *RESPParser) parseSimpleString() (RESPValue, error) {
	line, err := p.readLine()
	if err != nil {
		return RESPValue{}, err
	}
	return RESPValue{Type: SimpleString, Str: line}, nil
}

// parseError parses a RESP error
func (p *RESPParser) parseError() (RESPValue, error) {
	line, err := p.readLine()
	if err != nil {
		return RESPValue{}, err
	}
	return RESPValue{Type: Error, Str: line}, nil
}

// parseInteger parses a RESP integer
func (p *RESPParser) parseInteger() (RESPValue, error) {
	line, err := p.readLine()
	if err != nil {
		return RESPValue{}, err
	}
	num, err := strconv.Atoi(line)
	if err != nil {
		return RESPValue{}, err
	}
	return RESPValue{Type: Integer, Num: num}, nil
}

// readLine reads a line ending with \r\n
func (p *RESPParser) readLine() (string, error) {
	line, err := p.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(line, "\r\n"), nil
}

// RESPWriter handles writing RESP protocol messages
type RESPWriter struct {
	writer *bufio.Writer
}

// NewRESPWriter creates a new RESP writer
func NewRESPWriter(writer *bufio.Writer) *RESPWriter {
	return &RESPWriter{writer: writer}
}

// WriteSimpleString writes a RESP simple string
func (w *RESPWriter) WriteSimpleString(s string) error {
	_, err := w.writer.WriteString(fmt.Sprintf("+%s\r\n", s))
	if err != nil {
		return err
	}
	return w.writer.Flush()
}

// WriteError writes a RESP error
func (w *RESPWriter) WriteError(msg string) error {
	_, err := w.writer.WriteString(fmt.Sprintf("-%s\r\n", msg))
	if err != nil {
		return err
	}
	return w.writer.Flush()
}

// WriteBulkString writes a RESP bulk string
func (w *RESPWriter) WriteBulkString(s string) error {
	_, err := w.writer.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(s), s))
	if err != nil {
		return err
	}
	return w.writer.Flush()
}

// WriteInteger writes a RESP integer
func (w *RESPWriter) WriteInteger(num int) error {
	_, err := w.writer.WriteString(fmt.Sprintf(":%d\r\n", num))
	if err != nil {
		return err
	}
	return w.writer.Flush()
}

// WriteNullBulkString writes a RESP null bulk string
func (w *RESPWriter) WriteNullBulkString() error {
	_, err := w.writer.WriteString("$-1\r\n")
	if err != nil {
		return err
	}
	return w.writer.Flush()
}
