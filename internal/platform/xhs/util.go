package xhs

import (
	"math/big"
	"math/rand"
	"time"
)

func base36Encode(num *big.Int) string {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if num.Sign() == 0 {
		return "0"
	}
	
	var res string
	zero := big.NewInt(0)
	base := big.NewInt(36)
	
	n := new(big.Int).Set(num)
	if n.Sign() < 0 {
		res = "-"
		n.Abs(n)
	}
	
	var chars []byte
	mod := new(big.Int)
	
	for n.Cmp(zero) > 0 {
		n.DivMod(n, base, mod)
		chars = append(chars, alphabet[mod.Int64()])
	}
	
	// Reverse chars
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
	
	return res + string(chars)
}

func GetSearchId() string {
	// e = int(time.time() * 1000) << 64
	e := new(big.Int).SetInt64(time.Now().UnixMilli())
	e.Lsh(e, 64)
	
	// t = int(random.uniform(0, 2147483646))
	t := new(big.Int).SetInt64(int64(rand.Int31n(2147483646)))
	
	e.Add(e, t)
	return base36Encode(e)
}
