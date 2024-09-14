package snmp

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/davidjspooner/net-mapper/internal/asn1"
	"github.com/davidjspooner/net-mapper/internal/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
)

func init() {
	asn1.RegisterBinaryCodecs()
}

const v2c = 1

type protocol struct {
	community      string
	version        int
	bufferSize     int
	receiveTimeout time.Duration
}

type ProtocolOption func(p *protocol) error

func NewProtocol(options ...ProtocolOption) (Protocol, error) {
	p := &protocol{
		version:        -1,
		bufferSize:     4096,
		receiveTimeout: 2 * time.Second,
	}
	for _, option := range options {
		err := option(p)
		if err != nil {
			return nil, err
		}
	}
	if p.version == -1 {
		return nil, fmt.Errorf("version is required")
	}
	return p, nil
}

func (p *protocol) Dial(address string) (Connection, error) {
	parts := strings.Split(address, ":")
	port := 161
	var err error

	switch len(parts) {
	case 1:
		//all good nothing to see here
	case 2:
		port, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid port: %v", err)
		}
		address = parts[0]
	default:
		//TODO handle IPV6
		return nil, fmt.Errorf("invalid address: %s", address)
	}

	ip, err := net.ResolveIPAddr("ip", address)
	if err != nil {
		return nil, fmt.Errorf("error resolving address %s: %v", address, err)
	}

	udpAddr := net.UDPAddr{
		IP:   ip.IP,
		Port: port,
	}
	conn, err := net.DialUDP("udp", nil, &udpAddr)
	if err != nil {
		return nil, fmt.Errorf("error connecting to %s:%d : %v", address, port, err)
	}

	return &connection{protocol: p, conn: conn}, nil
}

func (p *protocol) DecodeFrame(frame []byte) (*Message, error) {
	message := Message{}
	_, err := asn1binary.Unmarshal(frame, &message)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling SNMP message: %v", err)
	}
	return &message, nil
}

func (p *protocol) EncodePDU(pType asn1core.Tag, pdu *PDU) ([]byte, error) {
	msg := Message{
		Version:   p.version,
		Community: p.community,
		PDU:       *pdu,
	}
	msg.PDU.Tag = pType
	msg.PDU.Class = asn1core.ClassContextSpecific
	bytes, err := asn1binary.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("error marshaling SNMP message: %v", err)
	}
	return bytes, nil
}

func WithV2(community string) ProtocolOption {
	return func(p *protocol) error {
		p.community = community
		p.version = v2c
		return nil
	}
}

func WithBufferSize(size int) ProtocolOption {
	return func(p *protocol) error {
		p.bufferSize = size
		return nil
	}
}

func WithReceiveTimeout(timeout time.Duration) ProtocolOption {
	return func(p *protocol) error {
		p.receiveTimeout = timeout
		return nil
	}
}
