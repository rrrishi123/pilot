package main

import "math"

// surfaceH replicates the JavaScript surfaceH(tx) from page.go (lines 77-79):
//
//	function rnd(n){ n=(n*1103515245+12345+seed)&0x7fffffff; return ((n>>16)&0x7fff)/0x7fff; }
//	function surfaceH(tx){ return Math.floor(GH + 4*Math.sin(tx*0.14+seed) + 2.5*Math.sin(tx*0.4) + 2*rnd(tx*13)-1); }
//
// where GH=42, seed=1337 (default).
func surfaceH(tx int, seed int64) int {
	// rnd — LCG matching JS: ((n*1103515245+12345+seed)&0x7fffffff) then ((n>>16)&0x7fff)/0x7fff
	rnd := func(n int) float64 {
		nn := (int64(n)*1103515245 + 12345 + seed) & 0x7fffffff
		return float64((nn>>16)&0x7fff) / 0x7fff
	}
	const GH = 42.0
	txf := float64(tx)
	h := GH + 4*math.Sin(txf*0.14+float64(seed)) + 2.5*math.Sin(txf*0.4) + 2*rnd(tx*13) - 1
	return int(math.Floor(h))
}
