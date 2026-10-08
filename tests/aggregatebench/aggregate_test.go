package aggregatebench

import "testing"

func initialBytes[T byteArray]() T {
	var value T
	for i := 0; i < len(value); i++ {
		value[i] = byte(i*17 + 3)
	}
	return value
}

func checkBytes[T byteArray](t *testing.T) {
	t.Helper()
	value := initialBytes[T]()
	original := value
	for _, seed := range []uint32{0, 1, 63, 64, 65, 255, 1024, 0xfffffffe} {
		got := editBytes(value, seed)
		for i := 0; i < len(value); i++ {
			want := value[i]
			if i == int(seed%uint32(len(value))) {
				want += byte(seed) | 1
			}
			if got[i] != want {
				t.Fatalf("seed %d index %d: got %d, want %d", seed, i, got[i], want)
			}
		}
		var want uint32
		for n := uint32(0); n < 4; n++ {
			s := seed + n
			i := int(s % uint32(len(value)))
			j := (i + len(value)/2) % len(value)
			want += uint32(value[i]+(byte(s)|1)) + uint32(value[j])
		}
		if got := serialBytes(value, seed); got != want {
			t.Fatalf("serial seed %d: got %d, want %d", seed, got, want)
		}
		if got := liveBytes(value, seed); got != want {
			t.Fatalf("live seed %d: got %d, want %d", seed, got, want)
		}
		if value != original {
			t.Fatal("callee changed the caller's array")
		}
	}
}

func TestBytes(t *testing.T) {
	if selectedBytes < 0 {
		t.Skip("shapes-only build")
	}
	if selectedBytes == 0 || selectedBytes == 16 {
		t.Run("16", checkBytes[[16]byte])
	}
	if selectedBytes == 0 || selectedBytes == 32 {
		t.Run("32", checkBytes[[32]byte])
	}
	if selectedBytes == 0 || selectedBytes == 64 {
		t.Run("64", checkBytes[[64]byte])
	}
	if selectedBytes == 0 || selectedBytes == 65 {
		t.Run("65", checkBytes[[65]byte])
	}
	if selectedBytes == 0 || selectedBytes == 128 {
		t.Run("128", checkBytes[[128]byte])
	}
	if selectedBytes == 0 || selectedBytes == 256 {
		t.Run("256", checkBytes[[256]byte])
	}
	if selectedBytes == 0 || selectedBytes == 512 {
		t.Run("512", checkBytes[[512]byte])
	}
	if selectedBytes == 0 || selectedBytes == 1024 {
		t.Run("1024", checkBytes[[1024]byte])
	}
	if selectedBytes == 0 || selectedBytes == 1025 {
		t.Run("1025", checkBytes[[1025]byte])
	}
}

func initialRecord() record {
	var value record
	for i := range value.Fields {
		value.Fields[i] = field{Text: "field", Values: []uint32{7, 11, 19}, Scale: uint32(i)}
	}
	value.ID = 23
	return value
}

func TestShapes(t *testing.T) {
	if selectedBytes > 0 {
		t.Skip("array-only build")
	}
	var words words64
	for i := range words {
		words[i] = uint64(i) << 40
	}
	gotWords := editWords(words, 65)
	if gotWords[1] != 1<<40+65 || words[1] != 1<<40 || readWords(gotWords, 65) != 34<<40+65 {
		t.Fatal("word value or copy mismatch")
	}
	var value scene
	value.Transforms[1][0] = point{X: 10, Y: 20, Z: 30, W: 40}
	gotScene := moveScene(value, 1)
	if readScene(gotScene, 1) != 101 || value.Transforms[1][0].X != 10 || value.ID != 0 {
		t.Fatal("scene value or copy mismatch")
	}
	r := initialRecord()
	var editors = []recordEditor{addEditor{}, xorEditor{}}
	for _, seed := range []uint32{0, 1, 64, 255} {
		i := seed % 12
		f := r.Fields[i]
		base := f.Scale + uint32(len(f.Text)) + f.Values[seed%3]
		wantAdd := r.ID + seed + base + (seed | 1)
		if got := readRecord(editRecord(r, seed), seed); got != wantAdd {
			t.Fatalf("record: got %d, want %d", got, wantAdd)
		}
		for j, editor := range editors {
			want := wantAdd
			if j == 1 {
				want = (r.ID ^ seed) + base
			}
			if got := invokeEditor(editor, r, seed); got != want {
				t.Fatalf("editor %d: got %d, want %d", j, got, want)
			}
		}
		if r.ID != 23 || r.Fields[i].Scale != i || r.Fields[i].Values[0] != 7 {
			t.Fatal("callee changed the caller's record")
		}
	}
}

func TestSequentialBlocks(t *testing.T) {
	if selectedBytes > 0 {
		t.Skip("array-only build")
	}
	for _, seed := range []byte{0, 1, 127, 255} {
		var want uint32
		for i := byte(0); i < 8; i++ {
			want += uint32((seed + i) * 2)
		}
		if got := sequentialBlocks(seed); got != want {
			t.Fatalf("seed %d: got %d, want %d", seed, got, want)
		}
	}
}
