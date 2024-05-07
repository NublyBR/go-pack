package pack

import "io"

// Get the size of buffer you need to store i as a VarUint
func SizeVarUint(i uint64) (n int) {
	// Every byte consumes 7 bits
	for {
		n += 1
		i >>= 7

		if i == 0 {
			return
		}
	}
}

// Puts a number in a buffer in the form of a VarUint, returning how many bytes were written
//
// Buffer must be at least SizeVarUint(i) bytes long, or 9 bytes for all possible uint64 values
func PutVarUint(i uint64, buf []byte) (n int) {
	// Bytes are represented with cddddddd
	// c = continuation flag
	// d = number data

	for i > 0x7f {
		buf[n] = byte(i&0x7f) | 0x80
		n++

		i >>= 7
	}

	buf[n] = byte(i & 0x7f)
	n++

	return
}

// Writes a number to a reader in the form of a VarUint, returning how many bytes were written
//
// Buffer must be at least SizeVarUint(i) bytes long, or 9 bytes for all possible uint64 values
func WriteVarUint(w io.Writer, i uint64, buf []byte) (n int, err error) {
	n = PutVarUint(i, buf)
	return w.Write(buf[:n])
}

// Gets a number in the form of a VarUint from a buffer, returning how many bytes were read
func GetVarUint(buf []byte) (i uint64, n int, err error) {
	ln := len(buf)

	if ln == 0 {
		err = io.EOF
		return
	}

	i = uint64(buf[n] & 0x7f)

	for buf[n]&0x80 == 0x80 {
		n++

		if n >= ln {
			err = io.EOF
			return
		}

		if n > 10 {
			err = ErrInvalidPackedUint
			return
		}

		i |= uint64(buf[n]&0x7f) << (7 * n)
	}

	n += 1

	return
}

// Gets a number in the form of a VarUint from a reader, returning how many bytes were read
//
// Buffer must be at least SizeVarUint(i) bytes long, or 9 bytes for all possible uint64 values
func ReadVarUint(r io.Reader, i *uint64, buf []byte) (int, error) {
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
