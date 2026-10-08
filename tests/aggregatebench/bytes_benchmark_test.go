//go:build !aggregatebench_flat

package aggregatebench

import "testing"

func benchmarkBytes[T byteArray](b *testing.B) {
	b.Run("Update", benchmarkBytesUpdate[T])
	b.Run("Read", benchmarkBytesRead[T])
	b.Run("Serial", benchmarkBytesSerial[T])
	b.Run("Live", benchmarkBytesLive[T])
}

func BenchmarkBytes(b *testing.B) {
	if selectedBytes < 0 {
		b.Skip("shapes-only build")
	}
	if selectedBytes == 0 || selectedBytes == 16 {
		b.Run("16", benchmarkBytes[[16]byte])
	}
	if selectedBytes == 0 || selectedBytes == 32 {
		b.Run("32", benchmarkBytes[[32]byte])
	}
	if selectedBytes == 0 || selectedBytes == 64 {
		b.Run("64", benchmarkBytes[[64]byte])
	}
	if selectedBytes == 0 || selectedBytes == 65 {
		b.Run("65", benchmarkBytes[[65]byte])
	}
	if selectedBytes == 0 || selectedBytes == 128 {
		b.Run("128", benchmarkBytes[[128]byte])
	}
	if selectedBytes == 0 || selectedBytes == 256 {
		b.Run("256", benchmarkBytes[[256]byte])
	}
	if selectedBytes == 0 || selectedBytes == 512 {
		b.Run("512", benchmarkBytes[[512]byte])
	}
	if selectedBytes == 0 || selectedBytes == 1024 {
		b.Run("1024", benchmarkBytes[[1024]byte])
	}
	if selectedBytes == 0 || selectedBytes == 1025 {
		b.Run("1025", benchmarkBytes[[1025]byte])
	}
}
