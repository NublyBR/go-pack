package pack

import "io"

func writeVarUint(w io.Writer, i uint64) (int, error) {
	var (
		total int
	)

	// Bytes are represented with cddddddd
	// c = continuation flag
	// d = number data

	for i > 0x7f {
		n, err := w.Write([]byte{byte(i&0x7f) | 0x80})
		if err != nil {
			return total, err
		}
		total += n

		i >>= 7
	}

	n, err := w.Write([]byte{byte(i & 0x7f)})
	if err != nil {
		return total, err
	}
	total += n

	return total, nil
}

func readVarUint(r io.Reader, i *uint64) (int, error) {
	var (
		total  int
		offset int
		result uint64
		buf    = []byte{0}
	)

	for {
		n, err := r.Read(buf)
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
