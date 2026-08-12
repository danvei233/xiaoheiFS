package money

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

var ErrInvalidAmount = fmt.Errorf("invalid amount")

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

	// Compute cents with big.Int so the x100+frac multiplication (and the
	// rounding increment) cannot silently wrap on int64 overflow.
	intBig, ok := new(big.Int).SetString(intPart, 10)
	if !ok {
		return 0, ErrInvalidAmount
	}
	cents := new(big.Int).Mul(intBig, big.NewInt(100))
	cents.Add(cents, big.NewInt(fracVal))

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
		cents.Add(cents, big.NewInt(1))
	}
	if !cents.IsInt64() {
		return 0, ErrInvalidAmount
	}
	return cents.Int64() * sign, nil
}

func ParseNumberStringToCents(raw string) (int64, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, ErrInvalidAmount
	}
	return ParseAmountToCents(raw)
}

func FormatCents(cents int64) string {
	sign := ""
	u := uint64(cents)
	if cents < 0 {
		sign = "-"
		// Negating MinInt64 overflows; compute the magnitude in uint64 instead.
		u = uint64(-(cents + 1)) + 1
	}
	integer := u / 100
	frac := u % 100
	return sign + strconv.FormatUint(integer, 10) + "." + leftPad2(int64(frac))
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
		return clampToInt64(quot)
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
	return clampToInt64(quot)
}

func clampToInt64(v *big.Int) int64 {
	if !v.IsInt64() {
		if v.Sign() > 0 {
			return math.MaxInt64
		}
		return math.MinInt64
	}
	return v.Int64()
}

func leftPad2(val int64) string {
	if val < 10 {
		return "0" + strconv.FormatInt(val, 10)
	}
	return strconv.FormatInt(val, 10)
}
