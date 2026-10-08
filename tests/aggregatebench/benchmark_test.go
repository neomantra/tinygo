package aggregatebench

import "testing"

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
