package money

import (
	"errors"
	"math/big"
	"strconv"
	"strings"
)

var ErrInvalidAmount = errors.New("invalid amount")

func ParseAmountToCents(input string) (int64, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return 0, ErrInvalidAmount
	}
	sign := int64(1)
	if s[0] == '-' {
		sign = -1
		s = strings.TrimSpace(s[1:])
	} else if s[0] == '+' {
		s = strings.TrimSpace(s[1:])
	}
	if s == "" {
		return 0, ErrInvalidAmount
	}
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	if intPart == "" {
		intPart = "0"
	}
	if _, err := strconv.ParseInt(intPart, 10, 64); err != nil {
		return 0, ErrInvalidAmount
	}
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	for _, ch := range frac {
		if ch < '0' || ch > '9' {
			return 0, ErrInvalidAmount
		}
	}
	for len(frac) < 3 {
		frac += "0"
	}
	roundDigit := int64(frac[2] - '0')
	rest := ""
	if len(frac) > 3 {
		rest = frac[3:]
	}
	frac = frac[:2]
	fracVal := int64(0)
	if frac != "" {
		if v, err := strconv.ParseInt(frac, 10, 64); err == nil {
			fracVal = v
		}
	}
	intVal, _ := strconv.ParseInt(intPart, 10, 64)
	cents := intVal*100 + fracVal
	roundUp := roundDigit > 5
	if roundDigit == 5 {
		roundUp = true
		if rest != "" {
			for _, ch := range rest {
				if ch != '0' {
					roundUp = true
					break
				}
			}
		}
	}
	if roundUp {
		cents++
	}
	return cents * sign, nil
}

func ParseNumberStringToCents(raw string) (int64, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, ErrInvalidAmount
	}
	return ParseAmountToCents(raw)
}

func FormatCents(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	integer := cents / 100
	frac := cents % 100
	return sign + strconv.FormatInt(integer, 10) + "." + leftPad2(frac)
}

func ProrateCents(cents, remain, total int64) int64 {
	if total == 0 {
		return 0
	}
	num := big.NewInt(cents)
	num.Mul(num, big.NewInt(remain))
	den := big.NewInt(total)
	quot, rem := new(big.Int).QuoRem(num, den, new(big.Int))
	if rem.Sign() == 0 {
		return quot.Int64()
	}
	absRem := new(big.Int).Abs(rem)
	doubleRem := new(big.Int).Mul(absRem, big.NewInt(2))
	if doubleRem.Cmp(new(big.Int).Abs(den)) >= 0 {
		if num.Sign() >= 0 {
			quot.Add(quot, big.NewInt(1))
		} else {
			quot.Sub(quot, big.NewInt(1))
		}
	}
	return quot.Int64()
}

func leftPad2(val int64) string {
	if val < 10 {
		return "0" + strconv.FormatInt(val, 10)
	}
	return strconv.FormatInt(val, 10)
}
