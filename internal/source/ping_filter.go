package source

import (
	"context"
	"encoding/binary"
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/davidjspooner/dsflow/pkg/job"
	"github.com/davidjspooner/net-mapper/internal/framework"
)

type pingFilter struct {
}

var _ Filter = (*pingFilter)(nil)

func init() {
	Register("ping", newPingFilter)
}

func newPingFilter(args framework.Config) (Source, error) {
	h := &pingFilter{}

	err := framework.CheckFields(args)
	if err != nil {
		return nil, err
	}

	return h, nil
}

// icmpMessage represents an ICMP message.
type icmpMessage struct {
	Type     uint8
	Code     uint8
	Checksum uint16
	ID       uint16
	Seq      uint16
}

// checksum calculates the ICMP checksum.
func checksum(data []byte) uint16 {
	var sum int32
	for i := 0; i < len(data)-1; i += 2 {
		sum += int32(binary.BigEndian.Uint16(data[i:]))
	}
	if len(data)%2 == 1 {
		sum += int32(data[len(data)-1])
	}
	sum = (sum >> 16) + (sum & 0xffff)
	sum += sum >> 16
	return uint16(^sum)
}

func (h *pingFilter) ping(ctx context.Context, host string, count int, interval time.Duration, timeout time.Duration) error {
	// Resolve the host address
	addr, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return err
	}

	// Create a raw socket
	conn, err := net.DialIP("ip4:icmp", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Create an ICMP echo request message
	icmp := icmpMessage{
		Type: 8,
		Code: 0,
		ID:   1,
		Seq:  1,
	}
	icmpBytes := make([]byte, 8)
	icmpBytes[0] = 8 // Echo request
	icmpBytes[1] = 0 // Code
	binary.BigEndian.PutUint16(icmpBytes[4:], icmp.ID)
	binary.BigEndian.PutUint16(icmpBytes[6:], 1) // Sequence number

	// Set the timeout
	deadline := time.Now().Add(timeout)
	conn.SetDeadline(deadline)

	// Send the ICMP echo request
	for id := 0; id < count; id++ {
		binary.BigEndian.PutUint16(icmpBytes[4:], uint16(id+1))
		icmp.Checksum = checksum(icmpBytes)
		binary.BigEndian.PutUint16(icmpBytes[2:], icmp.Checksum)
		_, err = conn.Write(icmpBytes)
		if err != nil {
			return err
		}
		time.Sleep(interval)
	}

	// Receive the ICMP echo reply
	reply := make([]byte, 20+8)
	_, err = conn.Read(reply)
	if err != nil {
		return err
	}

	// Check if the reply is an ICMP echo reply
	if reply[20] != 0 {
		return errors.New("no echo reply received")
	}

	return nil
}

func (h *pingFilter) pingShotgun(ctx context.Context, input HostList, count int, interval time.Duration, timeout time.Duration) (HostList, error) {
	output := make(HostList, 0, len(input))
	lock := sync.Mutex{}
	executer := job.NewExecuter[string](log.Default())
	executer.Start(ctx, 64, func(ctx context.Context, host string) error {
		err := h.ping(ctx, host, count, interval, timeout)
		if err != nil {
			return nil
		}
		log.Printf("Success %s \n", host)
		lock.Lock()
		defer lock.Unlock()
		output = append(output, host)
		return nil
	}, input)
	err := executer.WaitForCompletion()
	if err != nil {
		return nil, err
	}

	return output, nil
}

func (h *pingFilter) Filter(ctx context.Context, input HostList) (HostList, error) {
	output, err := h.pingShotgun(ctx, input, 1, time.Second, 2*time.Second)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (h *pingFilter) Kind() string {
	return "ping"
}
