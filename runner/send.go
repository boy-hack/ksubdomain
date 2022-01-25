package runner

import (
	"bytes"
	"encoding/gob"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"ksubdomain/core"
	"ksubdomain/core/gologger"
	"net"
	"sync/atomic"
	"time"
)

func (r *runner) sendCycle() {
	for {
		r.limit.Take()
		sender := <-r.sender
		if sender.Retry > r.maxRetry {
			r.hm.Del(sender.Domain)
			atomic.AddUint64(&r.faildIndex, 1)
			continue
		}
		sender.Retry += 1
		sender.Time = time.Now().Unix()
		var buff bytes.Buffer
		enc := gob.NewEncoder(&buff)
		err := enc.Encode(sender)
		if err != nil {
			continue
		}
		_ = r.hm.Set(sender.Domain, buff.Bytes())
		send(sender.Domain, sender.Dns, r.ether, r.dnsid, uint16(r.freeport), r.handle)
		atomic.AddUint64(&r.sentIndex, 1)
	}
}
func send(domain string, dnsname string, ether core.EthTable, dnsid uint16, freeport uint16, handle *pcap.Handle) {
	DstIp := net.ParseIP(dnsname).To4()
	eth := &layers.Ethernet{
		SrcMAC:       ether.SrcMac,
		DstMAC:       ether.DstMac,
		EthernetType: layers.EthernetTypeIPv4,
	}
	// Our IPv4 header
	ip := &layers.IPv4{
		Version:    4,
		IHL:        5,
		TOS:        0,
		Length:     0, // FIX
		Id:         0,
		Flags:      layers.IPv4DontFragment,
		FragOffset: 0,
		TTL:        255,
		Protocol:   layers.IPProtocolUDP,
		Checksum:   0,
		SrcIP:      ether.SrcIp,
		DstIP:      DstIp,
	}
	// Our UDP header
	udp := &layers.UDP{
		SrcPort: layers.UDPPort(freeport),
		DstPort: layers.UDPPort(53),
	}
	// Our DNS header
	dns := &layers.DNS{
		ID:      dnsid,
		QDCount: 1,
		//RD:      true, //递归查询标识
	}
	dns.Questions = append(dns.Questions,
		layers.DNSQuestion{
			Name:  []byte(domain),
			Type:  layers.DNSTypeA,
			Class: layers.DNSClassIN,
		})
	// Our UDP header
	_ = udp.SetNetworkLayerForChecksum(ip)
	buf := gopacket.NewSerializeBuffer()
	err := gopacket.SerializeLayers(
		buf,
		gopacket.SerializeOptions{
			ComputeChecksums: true, // automatically compute checksums
			FixLengths:       true,
		},
		eth, ip, udp, dns,
	)
	if err != nil {
		gologger.Warningf("SerializeLayers faild:%s\n", err.Error())
	}
	err = handle.WritePacketData(buf.Bytes())
	if err != nil {
		gologger.Warningf("WritePacketDate error:%s\n", err.Error())
	}
}
