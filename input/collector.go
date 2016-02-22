package input

import (
	"bufio"
	"crypto/tls"
	"expvar"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ekanite/ekanite/input/types"
)

var sequenceNumber int64
var stats = expvar.NewMap("input")

func init() {
	sequenceNumber = time.Now().UnixNano()
}

const (
	newlineTimeout = time.Duration(1000 * time.Millisecond)
	msgBufSize     = 256
)

// TCPCollector represents a network collector that accepts TCP packets.
type TCPCollector struct {
	iface     string
	delimiter types.Delimiter
	parser    types.Parser

	addr      net.Addr
	tlsConfig *tls.Config
}

// UDPCollector represents a network collector that accepts UDP packets.
type UDPCollector struct {
	addr   *net.UDPAddr
	parser types.Parser
}

func (s *TCPCollector) Addr() net.Addr {
	return s.addr
}

// NewCollector returns a network collector of the specified type, that will bind
// to the given inteface on Start(). If config is non-nil, a secure Collector will
// be returned. Secure Collectors require the protocol be TCP.
func NewCollector(proto string, builder types.Builder, iface string, tlsConfig *tls.Config) types.Collector {
	if strings.ToLower(proto) == "tcp" {
		return &TCPCollector{
			iface:     iface,
			delimiter: builder.NewDelimiter(),
			parser:    builder.NewParser(),
			tlsConfig: tlsConfig,
		}
	} else if strings.ToLower(proto) == "udp" {
		addr, err := net.ResolveUDPAddr("udp", iface)
		if err != nil {
			return nil
		}
		return &UDPCollector{addr: addr, parser: builder.NewParser()}
	}
	return nil
}

// Start instructs the TCPCollector to bind to the interface and accept connections.
func (s *TCPCollector) Start(c chan<- *types.Event) error {
	var ln net.Listener
	var err error
	if s.tlsConfig == nil {
		ln, err = net.Listen("tcp", s.iface)
	} else {
		ln, err = tls.Listen("tcp", s.iface, s.tlsConfig)
	}
	s.addr = ln.Addr()

	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			go s.handleConnection(conn, c)
		}
	}()
	return nil
}

func (s *TCPCollector) handleConnection(conn net.Conn, c chan<- *types.Event) {
	stats.Add("tcpConnections", 1)
	defer func() {
		stats.Add("tcpConnections", -1)
		conn.Close()
	}()

	reader := bufio.NewReader(conn)
	var log string
	var match bool

	for {
		conn.SetReadDeadline(time.Now().Add(newlineTimeout))
		b, err := reader.ReadByte()
		if err != nil {
			stats.Add("tcpConnReadError", 1)
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				stats.Add("tcpConnReadTimeout", 1)
				log, match = s.delimiter.Vestige()
			} else if err == io.EOF {
				stats.Add("tcpConnReadEOF", 1)
				log, match = s.delimiter.Vestige()
			} else {
				stats.Add("tcpConnUnrecoverError", 1)
				return
			}
		} else {
			stats.Add("tcpBytesRead", 1)
			log, match = s.delimiter.Push(b)
		}
		if match {
			stats.Add("tcpEventsRx", 1)
			c <- &types.Event{
				Text:          log,
				Parsed:        s.parser.Parse(log),
				ReceptionTime: time.Now().UTC(),
				Sequence:      atomic.AddInt64(&sequenceNumber, 1),
				SourceIP:      conn.RemoteAddr().String(),
			}
		}
	}
}

// Start instructs the UDPCollector to start reading packets from the interface.
func (s *UDPCollector) Start(c chan<- *types.Event) error {
	conn, err := net.ListenUDP("udp", s.addr)
	if err != nil {
		return err
	}

	go func() {
		buf := make([]byte, msgBufSize)
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			stats.Add("udpBytesRead", int64(n))
			if err != nil {
				continue
			}
			log := strings.Trim(string(buf[:n]), "\r\n")
			stats.Add("udpEventsRx", 1)
			c <- &types.Event{
				Text:          log,
				Parsed:        s.parser.Parse(log),
				ReceptionTime: time.Now().UTC(),
				Sequence:      atomic.AddInt64(&sequenceNumber, 1),
				SourceIP:      addr.String(),
			}
		}
	}()
	return nil
}

// Addr returns the net.Addr to which the UDP collector is bound.
func (s *UDPCollector) Addr() net.Addr {
	return s.addr
}
