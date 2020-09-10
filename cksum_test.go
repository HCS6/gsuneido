package main

import (
	"fmt"
	"hash/adler32"
	"hash/crc32"
	"math/rand"
	"testing"
)

var Sum uint32

const ndata = 1023

var Data = make([]byte, ndata)

func init() {
	for i := 0; i < ndata; i++ {
		Data[i] = byte(rand.Int31n(256))
	}
}

func BenchmarkAdler32(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Sum += adler32.Checksum(Data)
	}
}

func BenchmarkCrc32IEEE(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Sum += crc32.Checksum(Data, crc32.IEEETable)
	}
}

func BenchmarkCrc32Cast(b *testing.B) {
	table := crc32.MakeTable(crc32.Castagnoli)
	for i := 0; i < b.N; i++ {
		Sum += crc32.Checksum(Data, table)
	}
}

func TestDetection(*testing.T) {
	table := crc32.MakeTable(crc32.Castagnoli)
	var diff, lodiff, hidiff int
	for i := 0; i < 100000; i++ {
		a := crc32.Checksum(Data, table)
		for j := 0; j < 4; j++ {
			Data[rand.Int31n(ndata)] = byte(rand.Int31n(256))
		}
		b := crc32.Checksum(Data, table)
		if a != b {
			diff++
		}
		if uint16(a) != uint16(b) {
			lodiff++
		}
		if a>>16 != b>>16 {
			hidiff++
		}
	}
	fmt.Println("diff", diff, "lo", lodiff, "hi", hidiff)
}
