package aggregatebench

type byteArray interface {
	~[16]byte | ~[32]byte | ~[64]byte | ~[65]byte | ~[128]byte | ~[256]byte | ~[512]byte | ~[1024]byte | ~[1025]byte
}

//go:noinline
func editBytes[T byteArray](value T, seed uint32) T {
	i := int(seed % uint32(len(value)))
	value[i] += byte(seed) | 1
	return value
}

//go:noinline
func readBytes[T byteArray](value T, seed uint32) uint32 {
	i := int(seed % uint32(len(value)))
	j := (i + len(value)/2) % len(value)
	return uint32(value[i]) + uint32(value[j])
}

//go:noinline
func serialBytes[T byteArray](value T, seed uint32) uint32 {
	var sum uint32
	sum += readBytes(editBytes(value, seed), seed)
	sum += readBytes(editBytes(value, seed+1), seed+1)
	sum += readBytes(editBytes(value, seed+2), seed+2)
	sum += readBytes(editBytes(value, seed+3), seed+3)
	return sum
}

//go:noinline
func liveBytes[T byteArray](value T, seed uint32) uint32 {
	a := editBytes(value, seed)
	b := editBytes(value, seed+1)
	c := editBytes(value, seed+2)
	d := editBytes(value, seed+3)
	return readBytes(a, seed) + readBytes(b, seed+1) + readBytes(c, seed+2) + readBytes(d, seed+3)
}

type words64 [64]uint64

//go:noinline
func editWords(value words64, seed uint32) words64 {
	value[seed%64] += uint64(seed) | 1
	return value
}

//go:noinline
func readWords(value words64, seed uint32) uint64 {
	return value[seed%64] + value[(seed+32)%64]
}

type point struct {
	X, Y, Z, W float32
}

type matrix [4]point

type scene struct {
	Transforms [4]matrix
	ID         uint32
}

//go:noinline
func moveScene(value scene, seed uint32) scene {
	i := seed % 4
	j := (seed / 4) % 4
	value.Transforms[i][j].X += float32(seed & 15)
	value.Transforms[i][j].Y -= float32(seed & 7)
	value.ID += seed
	return value
}

//go:noinline
func readScene(value scene, seed uint32) float32 {
	p := value.Transforms[seed%4][(seed/4)%4]
	return p.X + p.Y + p.Z + p.W + float32(value.ID)
}

type field struct {
	Text   string
	Values []uint32
	Scale  uint32
}

type record struct {
	Fields [12]field
	ID     uint32
}

//go:noinline
func editRecord(value record, seed uint32) record {
	i := seed % 12
	value.Fields[i].Scale += seed | 1
	value.ID += seed
	return value
}

//go:noinline
func readRecord(value record, seed uint32) uint32 {
	f := value.Fields[seed%12]
	return value.ID + f.Scale + uint32(len(f.Text)) + f.Values[seed%uint32(len(f.Values))]
}

type recordEditor interface {
	Edit(record, uint32) record
}

type addEditor struct{}
type xorEditor struct{}

//go:noinline
func (addEditor) Edit(value record, seed uint32) record {
	return editRecord(value, seed)
}

//go:noinline
func (xorEditor) Edit(value record, seed uint32) record {
	value.ID ^= seed
	return value
}

//go:noinline
func invokeEditor(editor recordEditor, value record, seed uint32) uint32 {
	return readRecord(editor.Edit(value, seed), seed)
}

//go:noinline
func fillBlock(value *[512]byte, seed byte) uint32 {
	for i := range value {
		value[i] = seed + byte(i)
	}
	return uint32(value[int(seed)])
}

//go:noinline
func sequentialBlocks(seed byte) uint32 {
	var sum uint32
	sum += fillBlock(new([512]byte), seed)
	sum += fillBlock(new([512]byte), seed+1)
	sum += fillBlock(new([512]byte), seed+2)
	sum += fillBlock(new([512]byte), seed+3)
	sum += fillBlock(new([512]byte), seed+4)
	sum += fillBlock(new([512]byte), seed+5)
	sum += fillBlock(new([512]byte), seed+6)
	sum += fillBlock(new([512]byte), seed+7)
	return sum
}
