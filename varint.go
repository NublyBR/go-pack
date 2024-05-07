package pack

import (
	"io"
)

// Get the size of buffer you need to store i as a VarInt
func SizeVarInt(i int64) (n int) {
	// Ignore sign bit
	if i < 0 {
		i *= -1
	}

	// First byte consumes 6 bits
	i >>= 6
	n = 1

	// Remaining bytes consume 7 bits each
	for i != 0 {
		n += 1
		i >>= 7
	}

	return
}

// Puts a number in a buffer in the form of a VarInt, returning how many bytes were written
//
// Buffer must be at least SizeVarInt(i) bytes long, or 10 bytes for all possible int64 values
func PutVarInt(i int64, buf []byte) (n int) {
	n = 1

	// Write first byte with cndddddd
	// c = continuation flag
	// n = negative flag
	// d = number data
	// Following bytes use cddddddd
	{
		if i < 0 {
			i *= -1
			buf[0] = byte(i&0x3f) | 0x40
		} else {
			buf[0] = byte(i & 0x3f)
		}

		i >>= 6

		if i == 0 {
			return
		}

		buf[0] |= 0x80
	}

	for i > 0x7f {
		buf[n] = byte(i&0x7f) | 0x80
		n++

		i >>= 7
	}

	buf[n] = byte(i & 0x7f)
	n++

	return
}

// Writes a number to a reader in the form of a VarInt, returning how many bytes were written
//
// Buffer must be at least SizeVarInt(i) bytes long, or 10 bytes for all possible int64 values
func WriteVarInt(w io.Writer, i int64, buf []byte) (n int, err error) {
	n = PutVarInt(i, buf)
	return w.Write(buf[:n])
}

// Gets a number in the form of a VarInt from a buffer, returning how many bytes were read
func GetVarInt(buf []byte) (i int64, n int, err error) {
	var (
		ln = len(buf)

		isNegative bool
	)

	if ln == 0 {
		err = io.EOF
		return
	}

	isNegative = buf[0]&0x40 == 0x40
	i = int64(buf[0] & 0x3f)

	if buf[0]&0x80 == 0 {
		n = 1

		if isNegative {
			i *= -1
		}

		return
	}

	for {
		n++

		if n >= ln {
			err = io.EOF
			return
		}

		if n > 10 {
			err = ErrInvalidPackedInt
			return
		}

		i |= int64(buf[n]&0x7f) << (n*7 - 1)

		if buf[n]&0x80 == 0 {
			break
		}
	}

	n++

	if isNegative {
		i *= -1
	}

	return
}

// Gets a number in the form of a VarInt from a reader, returning how many bytes were read
//
// Buffer must be at least SizeVarInt(i) bytes long, or 10 bytes for all possible int64 values
func ReadVarInt(r io.Reader, i *int64, buf []byte) (int, error) {
	var (
		n   int
		err error

		total  int
		offset int
		result int64

		isNegative bool
	)

	{
		n, err = r.Read(buf[:1])
		if err != nil {
			return total, err
		}
		total += n

		result |= int64(buf[0]&0x3f) << offset
		offset += 6

		if buf[0]&0x40 == 0x40 { // n is set
			isNegative = true
		}

		if buf[0]&0x80 == 0 { // c is NOT set
			if isNegative {
				*i = -result
			} else {
				*i = result
			}

			return total, nil
		}
	}

	for {
		n, err = r.Read(buf[:1])
		if err != nil {
			return total, err
		}
		total += n

		result |= int64(buf[0]&0x7f) << offset
		offset += 7

		if buf[0]&0x80 == 0 {
			break
		}

		if offset > 64 {
			return total, ErrInvalidPackedInt
		}
	}

	if isNegative {
		*i = -result
	} else {
		*i = result
	}

	return total, nil
}
