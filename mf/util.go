package mf

import (
	"bytes"
	"io"
	"strings"
)

// NibbleU32 함수는 8개의 니블 배열을 부호 없는 32비트 정수형으로 변환합니다.
// 주의: 슬라이스의 길이가 8 미만일 경우 panic이 발생합니다.
func NibblesU32(nb []byte) uint32 {
	return uint32(nb[0]&0xf)<<28 | uint32(nb[1]&0xf)<<24 |
		uint32(nb[2]&0xf)<<20 | uint32(nb[3]&0xf)<<16 |
		uint32(nb[4]&0xf)<<12 | uint32(nb[5]&0xf)<<8 |
		uint32(nb[6]&0xf)<<4 | uint32(nb[7]&0xf)
}

// U32Nibbles 함수는 부호 없는 32비트 정수를 8개의 니블 배열로 변환합니다.
func U32Nibbles(n uint32) []byte {
	return []byte{
		byte(n>>28) & 0xf,
		byte(n>>24) & 0xf,
		byte(n>>20) & 0xf,
		byte(n>>16) & 0xf,
		byte(n>>12) & 0xf,
		byte(n>>8) & 0xf,
		byte(n>>4) & 0xf,
		byte(n) & 0xf,
	}
}

// BytesU32 함수는 바이트 슬라이스를 부호 없는 32비트 정수로 변환합니다
func BytesU32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// U32Bytes 함수는 부호 없는 32비트 정수를 바이트 슬라이스로 변환합니다.
func U32Bytes(n uint32) []byte {
	return []byte{byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}
}

// FromBfCode 함수는 Brainfuck 코드를 MinFuck 코드로 변환합니다.
// Brainfuck에는 사실상 memory address limit이 없기 때문에, 수동으로 지정해야 합니다.
func FromBfCode(bf string, mem uint32) (mf string) {
	fd := FileData{memsize: mem}
	nw := new(NibbleWriterOptimized)
	nw.NibbleWriter = new(NibbleWriter)
	for i := 0; i < 8; i++ {
		nw.Put(2)
	}
	nw.Flush()
	for _, b := range bf {
		op := FromBf(string(b))
		if op > 7 {
			continue
		}
		if op == 2 || op == 3 {
			for i := 0; i < 8; i++ {
				nw.Put(op)
			}
		} else {
			nw.Put(op)
		}
	}
	fd.code = nw.Nibbles
	mf = fd.String()
	return
}

// ToBfCode 함수는 MinFuck 코드를  Brainfuck 코드로 변환합니다.
func ToBfCode(mf string) (bf string) {
	bf = "MinFuck>>>>>>>>"

	meta, err := ReadFile(bytes.NewBufferString(mf))
	if err != nil {
		panic(err)
	}

	for i := uint32(0); i < meta.memsize; i++ {
		bf += strings.Repeat("+", int(i+1)) + ">>>>>>>>\n"
	}
	bf += strings.Repeat("<", int(meta.memsize)*8+8) + "\n"

	for _, mb := range meta.code {
		nb1, nb2 := (mb>>4)&0xf, mb&0xf
		bf += ToBf(nb1)
		bf += ToBf(nb2)
	}

	return
}

// FromBf 함수는 Brainfuck 코드를 MinFuck 코드로 변환합니다.
// TODO: 테스트 케이스 추가(BF 코드 아닌 경우 escape)
func FromBf(bf string) (mf byte) {
	switch bf {
	case "+":
		return 0
	case "-":
		return 1
	case ">":
		return 2
	case "<":
		return 3
	case "[":
		return 4
	case "]":
		return 5
	case ".":
		return 6
	case ",":
		return 7
	}
	return 255
}

// ToBf 함수는 MinFuck 코드를 BrainFuck 코드로 변환합니다.
func ToBf(mf byte) (p string) {
	switch mf {
	case 0:
		p = "+"
	case 1:
		p = "-"
	case 2:
		p = ">"
	case 3:
		p = "<"
	case 4:
		p = "["
	case 5:
		p = "]"
	case 6:
		p = "."
	case 7:
		p = ","
	}
	return p
}

// NibbleWriter 구조체는 니블코드를 byte slice로 변환해줍니다.
type NibbleWriter struct {
	Nibbles []byte
	odd     bool
}

// Put 메서드는 니블코드를 byte slice에 작성합니다.
func (n *NibbleWriter) Put(nb byte) {
	if len(n.Nibbles) == 0 {
		n.Nibbles = []byte{(nb & 0xf) << 4}
		n.odd = true
	} else if n.odd {
		n.Nibbles[len(n.Nibbles)-1] |= nb & 0xf
		n.odd = false
	} else {
		b := (nb & 0xf) << 4
		n.Nibbles = append(n.Nibbles, b)
		n.odd = true
	}
}

// NibbleWriterOptimized 구조체는 중복 니블코드를 압축해 byte slice에 작성합니다.
type NibbleWriterOptimized struct {
	*NibbleWriter
	buf byte
	cnt uint32
}

// Put 메셔드는 니블코드를 byte slice에 작성합니다.
func (n *NibbleWriterOptimized) Put(nb byte) {
	if n.buf != nb&0xf {
		n.Flush()
		n.buf = nb & 0xf
	}
	n.cnt++
}

// Flush 메서드는 버퍼에 있는 데이터를 byte slice에 작성하고 버퍼를 비웁니다.
func (n *NibbleWriterOptimized) Flush() {
	switch {
	case n.cnt == 0:
		return
	case n.cnt < 9 || n.buf < 4: // no compression
		for i := uint32(0); i < n.cnt; i++ {
			n.NibbleWriter.Put(n.buf)
		}
	default:
		if n.NibbleWriter.odd {
			n.NibbleWriter.Put(12)
		}
		n.NibbleWriter.Put(8 | n.buf)
		for _, nb := range U32Nibbles(n.cnt) {
			n.NibbleWriter.Put(nb)
		}
	}
	n.cnt = 0
}

// IOStream 구조체는 stdin/stdout을 에뮬레이션합니다.
// 주로 디버깅/에뮬레이션에 사용됩니다.
type IOStream struct {
	Stdin  string
	Stdout string
	offset uint64
}

// Read 메서드는 io.Reader 인터페이스를 구현합니다.
// MinFuck VM의 Stdin을 에뮬레이션합니다.
func (i *IOStream) Read(b []byte) (int, error) {
	if i.offset >= uint64(len(i.Stdin)) {
		return 0, io.EOF
	}
	n := copy(b, []byte(i.Stdin[i.offset:]))
	return n, nil
}

// Write 메서드는 io.Writer 인터페이스를 구현합니다.
// MinFuck VM의 Stdout을 에뮬레이션합니다.
func (i *IOStream) Write(b []byte) (int, error) {
	i.Stdout += string(b)
	return len(b), nil
}

func populateJump(bfcode string) []uint32 {

}
