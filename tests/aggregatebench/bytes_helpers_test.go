package aggregatebench

import "testing"

func benchmarkBytesUpdate[T byteArray](b *testing.B) {
	value := initialBytes[T]()
	current := value
	b.ReportAllocs()
	b.ResetTimer()
	var sum uint64
	for i := 0; i < b.N; i++ {
		current = editBytes(current, uint32(i))
		sum += uint64(readBytes(current, uint32(i)))
	}
	b.StopTimer()
	b.Logf("checksum %g", float64(sum))
}

func benchmarkBytesRead[T byteArray](b *testing.B) {
	value := initialBytes[T]()
	b.ReportAllocs()
	b.ResetTimer()
	var sum uint64
	for i := 0; i < b.N; i++ {
		sum += uint64(readBytes(value, uint32(i)))
	}
	b.StopTimer()
	b.Logf("checksum %g", float64(sum))
}

func benchmarkBytesSerial[T byteArray](b *testing.B) {
	value := initialBytes[T]()
	b.ReportAllocs()
	b.ResetTimer()
	var sum uint64
	for i := 0; i < b.N; i++ {
		sum += uint64(serialBytes(value, uint32(i)))
	}
	b.StopTimer()
	b.Logf("checksum %g", float64(sum))
}

func benchmarkBytesLive[T byteArray](b *testing.B) {
	value := initialBytes[T]()
	b.ReportAllocs()
	b.ResetTimer()
	var sum uint64
	for i := 0; i < b.N; i++ {
		sum += uint64(liveBytes(value, uint32(i)))
	}
	b.StopTimer()
	b.Logf("checksum %g", float64(sum))
}
