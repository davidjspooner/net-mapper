package main

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var ErrReassemblyNeeded = errors.New("reassembly needed")

type IPFrame struct {
	FrameNumber uint64
	IsFragment  bool
	IPProtocol  uint8
	SrcAddr     net.IPAddr
	DstAddr     net.IPAddr
	SrcPort     uint16
	DstPort     uint16
	Data        []byte
}

type IPFrameHandler interface {
	HandleIPFrame(frame *IPFrame) error
}

type IPFrameHandleFunc func(frame *IPFrame) error

func (f IPFrameHandleFunc) HandleIPFrame(frame *IPFrame) error {
	err := f(frame)
	if err != nil {
		return err
	}
	return nil
}

func PlaybackIpFramesFromFile(filename string, handler IPFrameHandler) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return PlaybackIpFramesFromStream(f, handler)
}

func PlaybackIpFramesFromStream(f *os.File, handler IPFrameHandler) error {
	r, err := pcap.OpenOfflineFile(f)
	if err != nil {
		return fmt.Errorf("failed to create pcap reader: %w", err)
	}

	packetSource := gopacket.NewPacketSource(r, r.LinkType())
	ipFrame := &IPFrame{}
	for packet := range packetSource.Packets() {
		ipFrame.FrameNumber++
		// Process each packet
		if ipV4 := packet.Layer(layers.LayerTypeIPv4); ipV4 != nil {
			ip := ipV4.(*layers.IPv4)
			ipFrame.IsFragment = ip.Flags&layers.IPv4MoreFragments != 0
			ipFrame.IPProtocol = uint8(ip.Protocol)
			ipFrame.SrcAddr = net.IPAddr{IP: ip.SrcIP}
			ipFrame.DstAddr = net.IPAddr{IP: ip.DstIP}
		} else if IPV6 := packet.Layer(layers.LayerTypeIPv6); IPV6 != nil {
			ip := IPV6.(*layers.IPv6)
			ipFrame.IsFragment = false
			ipFrame.IPProtocol = uint8(ip.NextHeader)
			ipFrame.SrcAddr = net.IPAddr{IP: ip.SrcIP}
			ipFrame.DstAddr = net.IPAddr{IP: ip.DstIP}
		} else {
			continue
		}
		if tcp := packet.Layer(layers.LayerTypeTCP); tcp != nil {
			tcp := tcp.(*layers.TCP)
			ipFrame.SrcPort = uint16(tcp.SrcPort)
			ipFrame.DstPort = uint16(tcp.DstPort)
			ipFrame.Data = tcp.Payload
		} else if udp := packet.Layer(layers.LayerTypeUDP); udp != nil {
			udp := udp.(*layers.UDP)
			ipFrame.SrcPort = uint16(udp.SrcPort)
			ipFrame.DstPort = uint16(udp.DstPort)
			ipFrame.Data = udp.Payload
		} else if icmp := packet.Layer(layers.LayerTypeICMPv4); icmp != nil {
			icmp := icmp.(*layers.ICMPv4)
			ipFrame.Data = icmp.LayerPayload()
			ipFrame.SrcPort = 0
			ipFrame.DstPort = 0
		} else if icmp := packet.Layer(layers.LayerTypeICMPv6); icmp != nil {
			icmp := icmp.(*layers.ICMPv6)
			ipFrame.Data = icmp.LayerPayload()
			ipFrame.SrcPort = 0
			ipFrame.DstPort = 0
		} else {
			continue
		}

		if err := handler.HandleIPFrame(ipFrame); err != nil {
			return fmt.Errorf("failed to handle IP frame: %w", err)
		}
	}

	return nil
}
