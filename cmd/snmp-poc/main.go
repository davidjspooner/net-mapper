package main

import (
	"context"
	"fmt"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
	"github.com/davidjspooner/net-mapper/pkg/snmp"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibdb"
)

func ReadAllMibs(ctx context.Context, dirname string) (*mibdb.Database, error) {

	db := mibdb.New()

	err := db.AddDirectory(dirname)
	if err != nil {
		return nil, err
	}
	err = db.CreateIndex(ctx)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func DecodeOIDAncVar(db *mibdb.Database, oid asn1go.OID, value asn1binary.Value) error {
	name, def, tail := db.FindOID(oid)
	if def == nil {
		fmt.Printf("             OID: %s Value: %v\n", oid.String(), value)
		return nil
	}
	if len(tail) == 1 && tail[0] == 0 {
		fmt.Printf("             OID: %s Value: %v\n", name, value)
		return nil
	}
	fmt.Printf("             OID: %s.%s Value: %v\n", name, tail.String(), value)
	return nil
}

func DecodeDump(filename string, db *mibdb.Database) error {
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
		// dump := hex.Dump(frame.Data)
		// for _, line := range strings.Split(dump, "\n") {
		// 	fmt.Printf("      %s\n", line)
		// }
		message, err := protocol.DecodeFrame(frame.Data)
		if err != nil {
			return fmt.Errorf("unmarshaling SNMP response: %w", err)
		}
		// switch message.PDU.Tag {
		// case snmp.GET:
		// 	fmt.Printf("      Method: GET\n")
		// case snmp.GET_NEXT:
		// 	fmt.Printf("      Method: GET_NEXT\n")
		// case snmp.GET_BULK:
		// 	fmt.Printf("      Method: GET_BULK\n")
		// case snmp.SET:
		// 	fmt.Printf("      Method: SET\n")
		// case snmp.TRAP:
		// 	fmt.Printf("      Method: TRAP\n")
		// case snmp.INFORM:
		// 	fmt.Printf("      Method: INFORM\n")
		// case snmp.RESPONSE:
		// 	fmt.Printf("      Method: RESPONSE\n")
		// default:
		// 	return fmt.Errorf("unexpected method: 0x%02X", message.PDU.Tag)
		// }
		// fmt.Printf("      Community: %s\n", message.Community)
		// fmt.Printf("      Version: %d\n", message.Version)
		// fmt.Printf("      RequestID: %d\n", message.PDU.RequestID)
		// if message.PDU.ErrorStatus > 0 {
		// 	fmt.Printf("      Error: %d\n", message.PDU.ErrorStatus)
		// }
		// if message.PDU.ErrorIndex > 0 {
		// 	fmt.Printf("      ErrorIndex: %d\n", message.PDU.ErrorIndex)
		// }
		for _, vb := range message.PDU.VarBinds {
			DecodeOIDAncVar(db, vb.OID, vb.Value)
		}
		return nil
	}))
	return err
}

func main() {
	ctx := context.Background()

	db, err := ReadAllMibs(ctx, "/mnt/homelab-atom/static/mib/")

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

	DecodeDump("/home/david/current/20240805_homelab/go/tool/dsnet-mapper/dumps/walk-20240914.pcap", db)
}
