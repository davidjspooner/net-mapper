package snmp

import (
	"fmt"
	"net"
	"time"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
)

type connection struct {
	protocol *protocol
	conn     *net.UDPConn
}

func (c *connection) Close() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return fmt.Errorf("connection already closed")
}

func (c *connection) Send(pType asn1core.Tag, pdu *PDU) error {

	bytes, err := c.protocol.EncodePDU(pType, pdu)
	if err != nil {
		return err
	}
	_, err = c.conn.Write(bytes)
	if err != nil {
		return fmt.Errorf("error sending SNMP message: %v", err)
	}
	return nil
}

func (c *connection) Receive() (*PDU, error) {
	buffer := make([]byte, 4096) //TODO: use a buffer pool
	c.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := c.conn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("error reading SNMP message: %v", err)
	}

	message, err := c.protocol.DecodeFrame(buffer[:n])
	if err != nil {
		return nil, err
	}
	return &message.PDU, nil
}
