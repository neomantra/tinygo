//go:build aggregatebench_flat

package aggregatebench

import "testing"

func BenchmarkBytes16Update(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 16 {
		b.Skip("unselected array size")
	}
	benchmarkBytesUpdate[[16]byte](b)
}

func BenchmarkBytes16Read(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 16 {
		b.Skip("unselected array size")
	}
	benchmarkBytesRead[[16]byte](b)
}

func BenchmarkBytes16Serial(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 16 {
		b.Skip("unselected array size")
	}
	benchmarkBytesSerial[[16]byte](b)
}

func BenchmarkBytes16Live(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 16 {
		b.Skip("unselected array size")
	}
	benchmarkBytesLive[[16]byte](b)
}

func BenchmarkBytes32Update(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 32 {
		b.Skip("unselected array size")
	}
	benchmarkBytesUpdate[[32]byte](b)
}

func BenchmarkBytes32Read(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 32 {
		b.Skip("unselected array size")
	}
	benchmarkBytesRead[[32]byte](b)
}

func BenchmarkBytes32Serial(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 32 {
		b.Skip("unselected array size")
	}
	benchmarkBytesSerial[[32]byte](b)
}

func BenchmarkBytes32Live(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 32 {
		b.Skip("unselected array size")
	}
	benchmarkBytesLive[[32]byte](b)
}

func BenchmarkBytes64Update(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 64 {
		b.Skip("unselected array size")
	}
	benchmarkBytesUpdate[[64]byte](b)
}

func BenchmarkBytes64Read(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 64 {
		b.Skip("unselected array size")
	}
	benchmarkBytesRead[[64]byte](b)
}

func BenchmarkBytes64Serial(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 64 {
		b.Skip("unselected array size")
	}
	benchmarkBytesSerial[[64]byte](b)
}

func BenchmarkBytes64Live(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 64 {
		b.Skip("unselected array size")
	}
	benchmarkBytesLive[[64]byte](b)
}

func BenchmarkBytes65Update(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 65 {
		b.Skip("unselected array size")
	}
	benchmarkBytesUpdate[[65]byte](b)
}

func BenchmarkBytes65Read(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 65 {
		b.Skip("unselected array size")
	}
	benchmarkBytesRead[[65]byte](b)
}

func BenchmarkBytes65Serial(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 65 {
		b.Skip("unselected array size")
	}
	benchmarkBytesSerial[[65]byte](b)
}

func BenchmarkBytes65Live(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 65 {
		b.Skip("unselected array size")
	}
	benchmarkBytesLive[[65]byte](b)
}

func BenchmarkBytes128Update(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 128 {
		b.Skip("unselected array size")
	}
	benchmarkBytesUpdate[[128]byte](b)
}

func BenchmarkBytes128Read(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 128 {
		b.Skip("unselected array size")
	}
	benchmarkBytesRead[[128]byte](b)
}

func BenchmarkBytes128Serial(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 128 {
		b.Skip("unselected array size")
	}
	benchmarkBytesSerial[[128]byte](b)
}

func BenchmarkBytes128Live(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 128 {
		b.Skip("unselected array size")
	}
	benchmarkBytesLive[[128]byte](b)
}

func BenchmarkBytes256Update(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 256 {
		b.Skip("unselected array size")
	}
	benchmarkBytesUpdate[[256]byte](b)
}

func BenchmarkBytes256Read(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 256 {
		b.Skip("unselected array size")
	}
	benchmarkBytesRead[[256]byte](b)
}

func BenchmarkBytes256Serial(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 256 {
		b.Skip("unselected array size")
	}
	benchmarkBytesSerial[[256]byte](b)
}

func BenchmarkBytes256Live(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 256 {
		b.Skip("unselected array size")
	}
	benchmarkBytesLive[[256]byte](b)
}

func BenchmarkBytes512Update(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 512 {
		b.Skip("unselected array size")
	}
	benchmarkBytesUpdate[[512]byte](b)
}

func BenchmarkBytes512Read(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 512 {
		b.Skip("unselected array size")
	}
	benchmarkBytesRead[[512]byte](b)
}

func BenchmarkBytes512Serial(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 512 {
		b.Skip("unselected array size")
	}
	benchmarkBytesSerial[[512]byte](b)
}

func BenchmarkBytes512Live(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 512 {
		b.Skip("unselected array size")
	}
	benchmarkBytesLive[[512]byte](b)
}

func BenchmarkBytes1024Update(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 1024 {
		b.Skip("unselected array size")
	}
	benchmarkBytesUpdate[[1024]byte](b)
}

func BenchmarkBytes1024Read(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 1024 {
		b.Skip("unselected array size")
	}
	benchmarkBytesRead[[1024]byte](b)
}

func BenchmarkBytes1024Serial(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 1024 {
		b.Skip("unselected array size")
	}
	benchmarkBytesSerial[[1024]byte](b)
}

func BenchmarkBytes1024Live(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 1024 {
		b.Skip("unselected array size")
	}
	benchmarkBytesLive[[1024]byte](b)
}

func BenchmarkBytes1025Update(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 1025 {
		b.Skip("unselected array size")
	}
	benchmarkBytesUpdate[[1025]byte](b)
}

func BenchmarkBytes1025Read(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 1025 {
		b.Skip("unselected array size")
	}
	benchmarkBytesRead[[1025]byte](b)
}

func BenchmarkBytes1025Serial(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 1025 {
		b.Skip("unselected array size")
	}
	benchmarkBytesSerial[[1025]byte](b)
}

func BenchmarkBytes1025Live(b *testing.B) {
	if selectedBytes != 0 && selectedBytes != 1025 {
		b.Skip("unselected array size")
	}
	benchmarkBytesLive[[1025]byte](b)
}
