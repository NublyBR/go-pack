package pack

import "io"

func writeVarInt(w io.Writer, i int64) (int, error) {
	var (
		total int
	)

	// Write first byte with cndddddd
	// c = continuation flag
	// n = negative flag
	// d = number data
	// Following bytes use cddddddd
	{
		var (
			flag byte
		)

		if i < 0 {
			i *= -1
			flag |= 0x40
		}

		if i > 0x3f {
			flag |= 0x80
		}

		n, err := w.Write([]byte{byte(i&0x3f) | flag})
		if err != nil {
			return total, err
		}
		total += n

		i >>= 6

		if i == 0 {
			return total, nil
		}
	}

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

func readVarInt(r io.Reader, i *int64) (int, error) {
	var (
		total  int
		offset int
		result int64
		buf    = []byte{0}

		isNegative bool
	)

	{
		n, err := r.Read(buf)
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
		n, err := r.Read(buf)
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
			return total, ErrInvalidPackedUint
		}
	}

	if isNegative {
		*i = -result
	} else {
		*i = result
	}

	return total, nil
}
