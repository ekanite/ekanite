package input

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"expvar"
	"io"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
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

// Collector specifies the interface all network collectors must implement.
type Collector interface {
	Start(chan<- *Event) error
	Addr() net.Addr
}

// TCPCollector represents a network collector that accepts TCP packets.
type TCPCollector struct {
	iface     string
	addr      net.Addr
	tlsConfig *tls.Config
	format    string
}

// UDPCollector represents a network collector that accepts UDP packets.
type UDPCollector struct {
	addr   *net.UDPAddr
	format string
}

func (s *TCPCollector) Addr() net.Addr {
	return s.addr
}

// NewCollector returns a network collector of the specified type, that will bind
// to the given inteface on Start(). If config is non-nil, a secure Collector will
// be returned. Secure Collectors require the protocol be TCP.
func NewCollector(proto string, iface string, tlsConfig *tls.Config, format string) Collector {
	if strings.ToLower(proto) == "tcp" {
		return &TCPCollector{
			iface:     iface,
			tlsConfig: tlsConfig,
			format:    format,
		}
	} else if strings.ToLower(proto) == "udp" {
		addr, err := net.ResolveUDPAddr("udp", iface)
		if err != nil {
			return nil
		}
		return &UDPCollector{addr: addr, format: format}
	}
	return nil
}

// Start instructs the TCPCollector to bind to the interface and accept connections.
func (s *TCPCollector) Start(c chan<- *Event) error {
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

func (s *TCPCollector) handleConnection(conn net.Conn, c chan<- *Event) {

	stats.Add("tcpConnections", 1)
	defer func() {
		stats.Add("tcpConnections", -1)
		conn.Close()
	}()

	parser := NewParser(s.format)
	reader := bufio.NewReader(conn)
	logBuff := bytes.NewBuffer([]byte(""))
	logLengthBuff := bytes.NewBuffer([]byte(""))
	var (
		expectLog bool
		logLength uint64
	)

	for {
		conn.SetReadDeadline(time.Now().Add(newlineTimeout))
		b, err := reader.ReadByte()
		if err != nil {
			stats.Add("tcpConnReadError", 1)
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				stats.Add("tcpConnReadTimeout", 1)
			} else if err == io.EOF {
				stats.Add("tcpConnReadEOF", 1)
			} else {
				stats.Add("tcpConnUnrecoverError", 1)
				return
			}
		} else {
			stats.Add("tcpBytesRead", 1)
			if expectLog {
				if logLength != 0 {
					logBuff.WriteByte(b)
					logLength--
				} else {
					ok, parsed := parser.Parse(logBuff.Bytes())
					if !ok {
						continue
					} else {
						stats.Add("tcpEventsRx", 1)
						c <- &Event{
							Text:          logBuff.String(),
							Parsed:        parsed,
							ReceptionTime: time.Now().UTC(),
							Sequence:      atomic.AddInt64(&sequenceNumber, 1),
							SourceIP:      conn.RemoteAddr().String(),
						}
					}
					logBuff.Reset()
					expectLog = false
				}
			} else {
				if b == ":"[0] {
					logLength, err = strconv.ParseUint(logLengthBuff.String(), 10, 64)
					if err != nil {
						stats.Add("tcpBytesLengthError", 1)
					} else {
						logLengthBuff.Reset()
					}
					expectLog = true
				} else {
					logBuff.WriteByte(b)
				}
			}
		}
	}
}

// Start instructs the UDPCollector to start reading packets from the interface.
func (s *UDPCollector) Start(c chan<- *Event) error {
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
			parser := NewParser(s.format)
			ok, parsed := parser.Parse([]byte(log))
			if !ok {
				continue
			} else {
				c <- &Event{
					Text:          log,
					Parsed:        parsed,
					ReceptionTime: time.Now().UTC(),
					Sequence:      atomic.AddInt64(&sequenceNumber, 1),
					SourceIP:      addr.String(),
				}
			}
		}
	}()
	return nil
}

// Addr returns the net.Addr to which the UDP collector is bound.
func (s *UDPCollector) Addr() net.Addr {
	return s.addr
}
