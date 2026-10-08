package aggregatebench

import "testing"

func benchmarkBytes[T byteArray](b *testing.B) {
	value := initialBytes[T]()
	b.Run("Update", func(b *testing.B) {
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
	})
	b.Run("Read", func(b *testing.B) {
		b.ReportAllocs()
		var sum uint64
		for i := 0; i < b.N; i++ {
			sum += uint64(readBytes(value, uint32(i)))
		}
		b.StopTimer()
		b.Logf("checksum %g", float64(sum))
	})
	b.Run("Serial", func(b *testing.B) {
		b.ReportAllocs()
		var sum uint64
		for i := 0; i < b.N; i++ {
			sum += uint64(serialBytes(value, uint32(i)))
		}
		b.StopTimer()
		b.Logf("checksum %g", float64(sum))
	})
	b.Run("Live", func(b *testing.B) {
		b.ReportAllocs()
		var sum uint64
		for i := 0; i < b.N; i++ {
			sum += uint64(liveBytes(value, uint32(i)))
		}
		b.StopTimer()
		b.Logf("checksum %g", float64(sum))
	})
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

func BenchmarkWords64(b *testing.B) {
	if selectedBytes > 0 {
		b.Skip("array-only build")
	}
	var value words64
	for i := range value {
		value[i] = uint64(i) << 40
	}
	b.ReportAllocs()
	b.ResetTimer()
	var sum uint64
	for i := 0; i < b.N; i++ {
		value = editWords(value, uint32(i))
		sum += readWords(value, uint32(i))
	}
	b.StopTimer()
	b.Logf("checksum %g", float64(sum))
}

func BenchmarkScene(b *testing.B) {
	if selectedBytes > 0 {
		b.Skip("array-only build")
	}
	var value scene
	for i := range value.Transforms {
		for j := range value.Transforms[i] {
			value.Transforms[i][j] = point{X: float32(i), Y: float32(j), Z: 2, W: 1}
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	var sum float32
	for i := 0; i < b.N; i++ {
		value = moveScene(value, uint32(i))
		sum += readScene(value, uint32(i))
	}
	b.StopTimer()
	b.Logf("checksum %g", float64(sum))
}

func BenchmarkRecord(b *testing.B) {
	if selectedBytes > 0 {
		b.Skip("array-only build")
	}
	value := initialRecord()
	b.ReportAllocs()
	b.ResetTimer()
	var sum uint64
	for i := 0; i < b.N; i++ {
		value = editRecord(value, uint32(i))
		sum += uint64(readRecord(value, uint32(i)))
	}
	b.StopTimer()
	b.Logf("checksum %g", float64(sum))
}

func BenchmarkRecordInterface(b *testing.B) {
	if selectedBytes > 0 {
		b.Skip("array-only build")
	}
	value := initialRecord()
	editors := [2]recordEditor{addEditor{}, xorEditor{}}
	b.ReportAllocs()
	b.ResetTimer()
	var sum uint64
	for i := 0; i < b.N; i++ {
		sum += uint64(invokeEditor(editors[i%2], value, uint32(i)))
	}
	b.StopTimer()
	b.Logf("checksum %g", float64(sum))
}

func BenchmarkSequentialBlocks(b *testing.B) {
	if selectedBytes > 0 {
		b.Skip("array-only build")
	}
	b.ReportAllocs()
	var sum uint64
	for i := 0; i < b.N; i++ {
		sum += uint64(sequentialBlocks(byte(i)))
	}
	b.StopTimer()
	b.Logf("checksum %g", float64(sum))
}
