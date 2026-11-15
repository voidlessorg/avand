package main

// پاسخ‌گوی mDNS ساده: avand.local را روی همه‌ی کارت‌های شبکه‌ی محلی اعلام می‌کند.

import (
	"encoding/binary"
	"net"
	"strings"
	"sync"
	"time"
)

var mdnsGroup = &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}

type mdnsIface struct {
	conn *net.UDPConn
	ip   net.IP
}

func (a *App) startMDNS() {
	name := a.cfg.Host + ".local"
	started := map[string]bool{}
	var mu sync.Mutex
	for {
		ifs, _ := net.Interfaces()
		for _, in := range ifs {
			if in.Flags&net.FlagUp == 0 || in.Flags&net.FlagMulticast == 0 || in.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, _ := in.Addrs()
			var ip net.IP
			for _, ad := range addrs {
				if ipn, ok := ad.(*net.IPNet); ok && ipn.IP.To4() != nil && isPrivate(ipn.IP) {
					ip = ipn.IP.To4()
				}
			}
			if ip == nil {
				continue
			}
			key := in.Name + "|" + ip.String()
			mu.Lock()
			done := started[key]
			started[key] = true
			mu.Unlock()
			if done {
				continue
			}
			ifc := in
			conn, err := net.ListenMulticastUDP("udp4", &ifc, mdnsGroup)
			if err != nil {
				mu.Lock()
				delete(started, key)
				mu.Unlock()
				continue
			}
			go serveMDNS(conn, name, ip)
		}
		time.Sleep(30 * time.Second)
	}
}

func serveMDNS(conn *net.UDPConn, name string, ip net.IP) {
	// اعلام اولیه
	_, _ = conn.WriteToUDP(buildMDNSAnswer(name, ip, 0, false), mdnsGroup)
	go func() {
		for range time.Tick(2 * time.Minute) {
			_, _ = conn.WriteToUDP(buildMDNSAnswer(name, ip, 0, false), mdnsGroup)
		}
	}()
	buf := make([]byte, 1500)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		id, unicast, ok := matchMDNSQuery(buf[:n], name)
		if !ok {
			continue
		}
		if src.Port != 5353 { // پرسش‌گر قدیمی (unicast DNS-over-mDNS)
			_, _ = conn.WriteToUDP(buildMDNSAnswer(name, ip, id, true), src)
		} else if unicast {
			_, _ = conn.WriteToUDP(buildMDNSAnswer(name, ip, 0, false), src)
		} else {
			_, _ = conn.WriteToUDP(buildMDNSAnswer(name, ip, 0, false), mdnsGroup)
		}
	}
}

func readName(b []byte, off int) (string, int, bool) {
	var parts []string
	for {
		if off >= len(b) {
			return "", 0, false
		}
		l := int(b[off])
		if l == 0 {
			return strings.Join(parts, "."), off + 1, true
		}
		if l&0xC0 != 0 { // فشرده‌سازی در بخش پرسش معمول نیست
			return "", 0, false
		}
		if off+1+l > len(b) {
			return "", 0, false
		}
		parts = append(parts, string(b[off+1:off+1+l]))
		off += 1 + l
	}
}

// آیا پرسشِ نوع A / ANY برای نام ما هست؟
func matchMDNSQuery(b []byte, name string) (id uint16, unicast bool, ok bool) {
	if len(b) < 12 || b[2]&0x80 != 0 { // پاسخ است نه پرسش
		return
	}
	id = binary.BigEndian.Uint16(b[0:2])
	qd := int(binary.BigEndian.Uint16(b[4:6]))
	off := 12
	for i := 0; i < qd; i++ {
		n, next, good := readName(b, off)
		if !good || next+4 > len(b) {
			return
		}
		qt, qc := binary.BigEndian.Uint16(b[next:]), binary.BigEndian.Uint16(b[next+2:])
		off = next + 4
		if strings.EqualFold(n, name) && (qt == 1 || qt == 255) && qc&0x7FFF == 1 {
			return id, qc&0x8000 != 0, true
		}
	}
	return
}

func buildMDNSAnswer(name string, ip net.IP, id uint16, legacy bool) []byte {
	out := make([]byte, 12)
	binary.BigEndian.PutUint16(out[0:], id)
	binary.BigEndian.PutUint16(out[2:], 0x8400) // پاسخ + authoritative
	binary.BigEndian.PutUint16(out[6:], 1)
	for _, p := range strings.Split(name, ".") {
		out = append(out, byte(len(p)))
		out = append(out, p...)
	}
	out = append(out, 0)
	cls := uint16(0x8001) // IN + cache-flush
	ttl := uint32(120)
	if legacy {
		cls, ttl = 1, 10
	}
	rr := make([]byte, 10)
	binary.BigEndian.PutUint16(rr[0:], 1)
	binary.BigEndian.PutUint16(rr[2:], cls)
	binary.BigEndian.PutUint32(rr[4:], ttl)
	binary.BigEndian.PutUint16(rr[8:], 4)
	out = append(out, rr...)
	return append(out, ip.To4()...)
}
