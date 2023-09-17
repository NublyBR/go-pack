package pack

import "io"

func writeVarUint(w io.Writer, i uint64, buf []byte) (int, error) {
	var (
		total int
	)

	// Bytes are represented with cddddddd
	// c = continuation flag
	// d = number data

	for i > 0x7f {
		buf[total] = byte(i&0x7f) | 0x80
		total++

		i >>= 7
	}

	buf[total] = byte(i & 0x7f)
	total++

	return w.Write(buf[:total])
}

func readVarUint(r io.Reader, i *uint64, buf []byte) (int, error) {
	var (
		total  int
		offset int
		result uint64
	)

	for {
		n, err := r.Read(buf[:1])
		if err != nil {
			return total, err
		}
		total += n

		result |= uint64(buf[0]&0x7f) << offset
		offset += 7

		if buf[0]&0x80 == 0 {
			break
		}

		if offset > 64 {
			return total, ErrInvalidPackedUint
		}
	}

	*i = result
	return total, nil
}
