package realname

import (
	"context"
	"strings"
	"time"
)

type IDCardCNProvider struct{}

func (p *IDCardCNProvider) Key() string {
	return "idcard_cn"
}

func (p *IDCardCNProvider) Name() string {
	return "China ID Card"
}

// provinceCodes is the set of valid GB/T 2260 province/region prefixes.
var provinceCodes = map[string]bool{
	"11": true, "12": true, "13": true, "14": true, "15": true,
	"21": true, "22": true, "23": true,
	"31": true, "32": true, "33": true, "34": true, "35": true, "36": true, "37": true,
	"41": true, "42": true, "43": true, "44": true, "45": true, "46": true,
	"50": true, "51": true, "52": true, "53": true, "54": true,
	"61": true, "62": true, "63": true, "64": true, "65": true,
	"71": true, "81": true, "82": true,
}

func (p *IDCardCNProvider) Verify(ctx context.Context, realName string, idNumber string) (bool, string, error) {
	realName = strings.TrimSpace(realName)
	idNumber = strings.TrimSpace(strings.ToUpper(idNumber))
	if realName == "" {
		return false, "real name required", nil
	}
	if len(idNumber) != 18 {
		return false, "id number length invalid", nil
	}
	if !validateIDCardDigits(idNumber) {
		return false, "id number format invalid", nil
	}
	if !validateIDCardRegion(idNumber) {
		return false, "id number region invalid", nil
	}
	if !validateIDCardBirth(idNumber) {
		return false, "id number birth date invalid", nil
	}
	if !validateIDCardChecksum(idNumber) {
		return false, "id number checksum invalid", nil
	}
	// A locally well-formed ID card is only input pre-validation: format checks
	// (region, birth date, checksum) cannot establish identity. Without an
	// authoritative provider the result must go to pending/manual review and
	// never grant verified, so an operator can issue verified only after review.
	return false, "pending", nil
}

// validateIDCardDigits checks the first 17 bytes are ASCII digits and the last
// byte is an ASCII digit or 'X'. Byte-wise checks reject double-width digits.
func validateIDCardDigits(id string) bool {
	if len(id) != 18 {
		return false
	}
	for i := 0; i < 17; i++ {
		if id[i] < '0' || id[i] > '9' {
			return false
		}
	}
	last := id[17]
	return last == 'X' || (last >= '0' && last <= '9')
}

// validateIDCardRegion validates the 6-digit region prefix against the
// province whitelist and rejects implausible zero-fill city/county parts.
func validateIDCardRegion(id string) bool {
	if len(id) < 6 {
		return false
	}
	region := id[0:6]
	if !provinceCodes[region[0:2]] {
		return false
	}
	// Reject codes like 110000 where both city and county are zero.
	return region[2:] != "0000"
}

func validateIDCardBirth(id string) bool {
	if len(id) < 14 {
		return false
	}
	birth := id[6:14]
	t, err := time.Parse("20060102", birth)
	if err != nil {
		return false
	}
	if t.Year() < 1900 {
		return false
	}
	if t.After(time.Now()) {
		return false
	}
	return true
}

func validateIDCardChecksum(id string) bool {
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	mapping := []byte{'1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'}
	sum := 0
	for i := 0; i < 17; i++ {
		sum += int(id[i]-'0') * weights[i]
	}
	check := mapping[sum%11]
	return id[17] == check
}
