package realname

import (
	"context"
	"strings"
	"time"
	"unicode"
)

type IDCardCNProvider struct{}

func (p *IDCardCNProvider) Key() string {
	return "idcard_cn"
}

func (p *IDCardCNProvider) Name() string {
	return "China ID Card"
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
	if !validateIDCardBirth(idNumber) {
		return false, "id number birth date invalid", nil
	}
	if !validateIDCardChecksum(idNumber) {
		return false, "id number checksum invalid", nil
	}
	return true, "", nil
}

func validateIDCardDigits(id string) bool {
	for i, r := range id {
		if i == 17 && (r == 'X' || unicode.IsDigit(r)) {
			continue
		}
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func validateIDCardBirth(id string) bool {
	if len(id) < 14 {
		return false
	}
	birth := id[6:14]
	_, err := time.Parse("20060102", birth)
	return err == nil
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
