package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
	"github.com/davidjspooner/net-mapper/pkg/snmp"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibdb"
)

func ReadAllMibs(ctx context.Context, dirname string) error {

	mibDB := mibdb.New()

	err := mibDB.AddDirectory(dirname)
	if err != nil {
		return err
	}
	err = mibDB.CreateIndex(ctx)
	return err
}

func DecodeDump(filename string) error {
	community := "public"
	//oid := asn1.ObjectIdentifier{1, 3, 6, 1, 2, 1, 1, 1, 0} // Replace with your OID

	protocol, err := snmp.NewProtocol(snmp.WithV2(community))
	if err != nil {
		fmt.Printf("Error creating SNMP protocol: %v\n", err)
		return err
	}

	err = PlaybackIpFramesFromFile("/home/david/current/20240805_homelab/go/tool/dsnet-mapper/dumps/walk-20240914.pcap", IPFrameHandleFunc(func(frame *IPFrame) error {
		if frame.IsFragment {
			return ErrReassemblyNeeded
		}
		if frame.IPProtocol != 17 { //udp
			return nil
		}
		fmt.Printf("Frame: %d Src: %s:%d, Dst: %s:%d\n", frame.FrameNumber, frame.SrcAddr.IP, frame.SrcPort, frame.DstAddr.IP, frame.DstPort)
		dump := hex.Dump(frame.Data)
		for _, line := range strings.Split(dump, "\n") {
			fmt.Printf("      %s\n", line)
		}
		message, err := protocol.DecodeFrame(frame.Data)
		if err != nil {
			return fmt.Errorf("unmarshaling SNMP response: %w", err)
		}
		switch message.PDU.Tag {
		case snmp.GET:
			fmt.Printf("      Method: GET\n")
		case snmp.GET_NEXT:
			fmt.Printf("      Method: GET_NEXT\n")
		case snmp.GET_BULK:
			fmt.Printf("      Method: GET_BULK\n")
		case snmp.SET:
			fmt.Printf("      Method: SET\n")
		case snmp.TRAP:
			fmt.Printf("      Method: TRAP\n")
		case snmp.INFORM:
			fmt.Printf("      Method: INFORM\n")
		case snmp.RESPONSE:
			fmt.Printf("      Method: RESPONSE\n")
		default:
			return fmt.Errorf("unexpected method: 0x%02X", message.PDU.Tag)
		}
		fmt.Printf("      Community: %s\n", message.Community)
		fmt.Printf("      Version: %d\n", message.Version)
		fmt.Printf("      RequestID: %d\n", message.PDU.RequestID)
		if message.PDU.ErrorStatus > 0 {
			fmt.Printf("      Error: %d\n", message.PDU.ErrorStatus)
		}
		if message.PDU.ErrorIndex > 0 {
			fmt.Printf("      ErrorIndex: %d\n", message.PDU.ErrorIndex)
		}
		for _, vb := range message.PDU.VarBinds {
			fmt.Printf("             OID: %s, Value: %v\n", vb.OID.String(), vb.Value)
		}
		return nil
	}))
	return err
}

func main() {
	ctx := context.Background()

	err := ReadAllMibs(ctx, "/mnt/homelab-atom/static/mib/")

	if err != nil {
		errors, multiError := err.(asn1core.ErrorList)
		if multiError {
			for _, e := range errors {
				fmt.Printf("Error: %v\n", e)
			}
		} else {
			fmt.Printf("Error: %v\n", err)
		}
	}

	//	for _, vb := range response.VarBinds {
	//		fmt.Printf("OID: %s, Value: %v\n", vb.OID, vb.Value)
	//	}
}
