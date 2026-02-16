package http

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/oschwald/geoip2-golang"
)

type GeoResolver interface {
	Resolve(ctx context.Context, ip, mmdbPath string) (city string, tz string, err error)
}

type MMDBGeoResolver struct {
	mu       sync.RWMutex
	dbPath   string
	dbReader *geoip2.Reader
}

func NewMMDBGeoResolver() *MMDBGeoResolver {
	return &MMDBGeoResolver{}
}

func (r *MMDBGeoResolver) Resolve(_ context.Context, rawIP, mmdbPath string) (string, string, error) {
	path := strings.TrimSpace(mmdbPath)
	if path == "" {
		return "", "", errors.New("geoip mmdb path empty")
	}
	ip := net.ParseIP(strings.TrimSpace(rawIP))
	if ip == nil {
		return "", "", errors.New("invalid ip")
	}
	if isNonPublicIP(ip) {
		return "", "", errors.New("non-public ip")
	}
	db, err := r.getOrOpen(path)
	if err != nil {
		return "", "", err
	}
	record, err := db.City(ip)
	if err != nil {
		return "", "", err
	}
	city := strings.TrimSpace(record.City.Names["zh-CN"])
	if city == "" {
		city = strings.TrimSpace(record.City.Names["en"])
	}
	if city == "" {
		city = strings.TrimSpace(record.Country.Names["zh-CN"])
	}
	if city == "" {
		city = strings.TrimSpace(record.Country.Names["en"])
	}
	tz := formatGMTFromIANA(strings.TrimSpace(record.Location.TimeZone))
	return city, tz, nil
}

func (r *MMDBGeoResolver) getOrOpen(path string) (*geoip2.Reader, error) {
	r.mu.RLock()
	if r.dbReader != nil && r.dbPath == path {
		db := r.dbReader
		r.mu.RUnlock()
		return db, nil
	}
	r.mu.RUnlock()

	newDB, err := geoip2.Open(path)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.dbReader != nil && r.dbPath == path {
		_ = newDB.Close()
		return r.dbReader, nil
	}
	old := r.dbReader
	r.dbReader = newDB
	r.dbPath = path
	if old != nil {
		_ = old.Close()
	}
	return r.dbReader, nil
}

func isNonPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		switch {
		case v4[0] == 10:
			return true
		case v4[0] == 172 && v4[1] >= 16 && v4[1] <= 31:
			return true
		case v4[0] == 192 && v4[1] == 168:
			return true
		case v4[0] == 127:
			return true
		case v4[0] == 169 && v4[1] == 254:
			return true
		}
		return false
	}
	return ip.IsPrivate()
}

func formatGMTFromIANA(iana string) string {
	if strings.TrimSpace(iana) == "" {
		return ""
	}
	loc, err := time.LoadLocation(iana)
	if err != nil {
		return ""
	}
	_, offset := time.Now().In(loc).Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	mins := (offset % 3600) / 60
	return "GMT" + sign + twoDigits(hours) + ":" + twoDigits(mins)
}

func twoDigits(v int) string {
	if v >= 10 {
		return strconv.Itoa(v)
	}
	return "0" + strconv.Itoa(v)
}
