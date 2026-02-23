package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type DNSHandler struct {
	Storage *MemoryStorage
	Metrics *Metrics
}

func (h *DNSHandler) recordDNSMetrics(qtype, geoTag, status string, d time.Duration) {
	if h.Metrics == nil {
		return
	}
	h.Metrics.DNSQueriesTotal.WithLabelValues(qtype, geoTag, status).Inc()
	h.Metrics.DNSQueryDuration.WithLabelValues(qtype, geoTag).Observe(d.Seconds())
}

func (h *DNSHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	start := time.Now()
	msg := new(dns.Msg)
	msg.SetReply(r)

	clientIP, _, _ := net.SplitHostPort(w.RemoteAddr().String())
	parsedIP := net.ParseIP(clientIP)
	geoTag := h.Storage.GetGeoTag(clientIP)

	for _, question := range r.Question {
		queryName := strings.ToLower(question.Name)
		qtype := dns.TypeToString[question.Qtype]
		if qtype == "" {
			qtype = strconv.Itoa(int(question.Qtype))
		}

		// AXFR доступен только для IP из whitelist
		if question.Qtype == dns.TypeAXFR {
			if !h.Storage.IsAllowed(parsedIP) {
				msg.Rcode = dns.RcodeRefused
				w.WriteMsg(msg)
				h.recordDNSMetrics(qtype, geoTag, "refused", time.Since(start))
				return
			}
			h.handleAXFR(w, r, queryName, geoTag)
			h.recordDNSMetrics(qtype, geoTag, "answered", time.Since(start))
			return
		}

		records := h.Storage.GetRecordsForQuery(queryName, geoTag)

		if len(records) > 0 {
			msg.Authoritative = true
			for _, rec := range records {
				rr := buildRR(rec, question.Qtype)
				if rr != nil {
					msg.Answer = append(msg.Answer, rr)
				}
			}
			w.WriteMsg(msg)
			h.recordDNSMetrics(qtype, geoTag, "answered", time.Since(start))
			return
		} else if question.Qtype == dns.TypeSOA {
			soa := h.Storage.GetSOA(queryName, geoTag)
			if soa != nil {
				rr, _ := dns.NewRR(fmt.Sprintf("%s %d IN SOA %s %s %d %d %d %d %d",
					queryName, soa.MinTTL, soa.Ns, soa.Mbox, soa.Serial, soa.Refresh, soa.Retry, soa.Expire, soa.MinTTL))
				msg.Answer = append(msg.Answer, rr)
			}
			w.WriteMsg(msg)
			h.recordDNSMetrics(qtype, geoTag, "answered", time.Since(start))
			return
		} else if h.Storage.IsAllowed(parsedIP) {
			msg.RecursionAvailable = true
			c := new(dns.Client)
			in, _, err := c.Exchange(r, "8.8.8.8:53")
			if err == nil {
				w.WriteMsg(in)
				h.recordDNSMetrics(qtype, geoTag, "recursed", time.Since(start))
				return
			}
		} else {
			msg.Rcode = dns.RcodeRefused
			w.WriteMsg(msg)
			h.recordDNSMetrics(qtype, geoTag, "refused", time.Since(start))
			return
		}
	}

	w.WriteMsg(msg)
	h.recordDNSMetrics("unknown", geoTag, "nxdomain", time.Since(start))
}

// buildRR конвертирует ResourceRecord в dns.RR нужного типа.
// Для MX формат Value: "10 mail.example.com."
// Для остальных Value содержит непосредственно данные.
func buildRR(rec ResourceRecord, qtype uint16) dns.RR {
	recType := strings.ToUpper(rec.Type)

	switch {
	case recType == "A" && qtype == dns.TypeA:
		rr := &dns.A{
			Hdr: dns.RR_Header{Name: rec.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: rec.TTL},
			A:   net.ParseIP(rec.Value).To4(),
		}
		if rr.A == nil {
			return nil
		}
		return rr

	case recType == "AAAA" && qtype == dns.TypeAAAA:
		rr := &dns.AAAA{
			Hdr:  dns.RR_Header{Name: rec.Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: rec.TTL},
			AAAA: net.ParseIP(rec.Value).To16(),
		}
		if rr.AAAA == nil {
			return nil
		}
		return rr

	case recType == "CNAME" && qtype == dns.TypeCNAME:
		return &dns.CNAME{
			Hdr:    dns.RR_Header{Name: rec.Name, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: rec.TTL},
			Target: dns.Fqdn(rec.Value),
		}

	case recType == "NS" && qtype == dns.TypeNS:
		return &dns.NS{
			Hdr: dns.RR_Header{Name: rec.Name, Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: rec.TTL},
			Ns:  dns.Fqdn(rec.Value),
		}

	case recType == "TXT" && qtype == dns.TypeTXT:
		return &dns.TXT{
			Hdr: dns.RR_Header{Name: rec.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: rec.TTL},
			Txt: []string{rec.Value},
		}

	case recType == "MX" && qtype == dns.TypeMX:
		// Value format: "10 mail.example.com."
		parts := strings.SplitN(rec.Value, " ", 2)
		if len(parts) != 2 {
			return nil
		}
		pref, err := strconv.ParseUint(parts[0], 10, 16)
		if err != nil {
			return nil
		}
		return &dns.MX{
			Hdr:        dns.RR_Header{Name: rec.Name, Rrtype: dns.TypeMX, Class: dns.ClassINET, Ttl: rec.TTL},
			Preference: uint16(pref),
			Mx:         dns.Fqdn(parts[1]),
		}
	}

	return nil
}

// handleAXFR отдаёт полный дамп зоны (SOA + все записи + SOA).
// Вызывается только для IP из whitelist.
func (h *DNSHandler) handleAXFR(w dns.ResponseWriter, r *dns.Msg, origin, geoTag string) {
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true

	zone := h.Storage.GetZone(origin, geoTag)
	if zone == nil {
		msg.Rcode = dns.RcodeNotAuth
		w.WriteMsg(msg)
		return
	}

	soaStr := fmt.Sprintf("%s %d IN SOA %s %s %d %d %d %d %d",
		origin, zone.SOA.MinTTL, zone.SOA.Ns, zone.SOA.Mbox,
		zone.SOA.Serial, zone.SOA.Refresh, zone.SOA.Retry, zone.SOA.Expire, zone.SOA.MinTTL)
	soaRR, _ := dns.NewRR(soaStr)

	msg.Answer = append(msg.Answer, soaRR)
	for _, rec := range zone.Records {
		if rr := buildRRAny(rec); rr != nil {
			msg.Answer = append(msg.Answer, rr)
		}
	}
	msg.Answer = append(msg.Answer, soaRR) // AXFR завершается повторной SOA

	w.WriteMsg(msg)
}

// buildRRAny строит dns.RR без фильтрации по типу запроса (для AXFR).
func buildRRAny(rec ResourceRecord) dns.RR {
	// Переиспользуем buildRR, передавая нативный тип записи
	typeMap := map[string]uint16{
		"A":     dns.TypeA,
		"AAAA":  dns.TypeAAAA,
		"CNAME": dns.TypeCNAME,
		"NS":    dns.TypeNS,
		"TXT":   dns.TypeTXT,
		"MX":    dns.TypeMX,
	}
	qtype, ok := typeMap[strings.ToUpper(rec.Type)]
	if !ok {
		return nil
	}
	return buildRR(rec, qtype)
}
