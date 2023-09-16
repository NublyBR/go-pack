package pack

import (
	"encoding/binary"
	"io"
	"math"
	"reflect"
)

type Packer interface {
	// Encode object into stream
	//
	// Note: If a pointer is given to the encoder (*obj), then the decoder needs to receive a pointer to a pointer (**obj)
	// (unless object mode is enabled in the options)
	Encode(data any) error

	// Total bytes written to the underlying stream
	BytesWritten() uint64
}

type packer struct {
	writer  io.Writer
	written uint64

	objects Objects
}

func NewPacker(writer io.Writer, options ...Options) Packer {
	p := &packer{writer: writer}

	for _, opt := range options {
		if opt.WithObjects != nil {
			p.objects = opt.WithObjects
		}
	}

	return p
}

func (p *packer) Encode(data any) error {

	if p.objects != nil {

		oid, exists := p.objects.GetID(data)
		if !exists {
			return &ErrNotDefined{typ: reflect.TypeOf(data)}
		}

		n, err := writeVarUint(p.writer, uint64(oid))
		p.written += uint64(n)
		if err != nil {
			return err
		}

		read, err := p.encode(data, packerInfo{})
		p.written += uint64(read)

		return err

	}

	n, err := p.encode(data, packerInfo{})
	p.written += uint64(n)

	return err
}

func (p *packer) BytesWritten() uint64 {
	return p.written
}

func (p *packer) encodeBytes(data []byte, inf packerInfo) (int, error) {
	var total int

	if inf.maxSize > 0 && uint64(len(data)) > inf.maxSize {
		return total, &ErrDataTooLarge{typ: reflect.TypeOf(data), max: inf.maxSize, size: uint64(len(data))}
	}

	n, err := writeVarUint(p.writer, uint64(len(data)))
	total += n
	if err != nil {
		return total, err
	}

	n, err = p.writer.Write(data)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (p *packer) encodeType(typ reflect.Type) (int, error) {

	var (
		total int

		kind reflect.Kind
	)

	if typ == nil {
		kind = 0xff
	} else {
		kind = typ.Kind()
	}

	if !canEncodeInInterface[kind] {
		return total, &ErrCantUseInInterfaceMode{kind: kind, typ: typ}
	}

	n, err := p.writer.Write([]byte{byte(kind)})
	total += n
	if err != nil {
		return total, err
	}

	switch kind {
	case reflect.Pointer:
		n, err = p.encodeType(typ.Elem())
		total += n
		if err != nil {
			return total, err
		}

	case reflect.Array:
		n, err := writeVarUint(p.writer, uint64(typ.Len()))
		total += n
		if err != nil {
			return total, err
		}

		n, err = p.encodeType(typ.Elem())
		total += n
		if err != nil {
			return total, err
		}

	case reflect.Map:
		n, err = p.encodeType(typ.Key())
		total += n
		if err != nil {
			return total, err
		}

		n, err = p.encodeType(typ.Elem())
		total += n
		if err != nil {
			return total, err
		}

	case reflect.Slice:
		n, err = p.encodeType(typ.Elem())
		total += n
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

func (p *packer) encode(data any, info packerInfo) (int, error) {
	if info.ignore {
		return 0, nil
	}

	var (
		typ = reflect.TypeOf(data)
		val = reflect.ValueOf(data)

		total int

		buffer [8]byte
	)

	if info.markType {
		n, err := p.encodeType(typ)
		total += n
		if err != nil {
			return total, err
		}

		if typ == nil {
			return total, err
		}
	}

	if typ == nil {
		return total, ErrNil
	}

	for typ.Kind() == reflect.Pointer {
		if val.IsNil() {
			n, err := p.writer.Write([]byte{0})
			total += n
			if err != nil {
				return total, err
			}
			return total, nil
		}

		n, err := p.writer.Write([]byte{1})
		total += n
		if err != nil {
			return total, err
		}

		typ = typ.Elem()
		val = val.Elem()
	}

	switch typ.Kind() {
	case reflect.Int8, reflect.Uint8, reflect.Bool:
		var (
			n   int
			err error
		)

		switch cast := val.Interface().(type) {
		case uint8:
			n, err = p.writer.Write([]byte{byte(cast)})
		case int8:
			n, err = p.writer.Write([]byte{byte(cast)})
		case bool:
			if cast {
				n, err = p.writer.Write([]byte{1})
				total += n
				return total, err
			}

			n, err = p.writer.Write([]byte{0})
			total += n
			return total, err
		}

		total += n
		return total, err

	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
		var value int64

		switch cast := val.Interface().(type) {
		case int:
			value = int64(cast)
		case int16:
			value = int64(cast)
		case int32:
			value = int64(cast)
		case int64:
			value = int64(cast)
		}

		n, err := writeVarInt(p.writer, value)
		total += n
		return total, err

	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		var value uint64

		switch cast := val.Interface().(type) {
		case uint:
			value = uint64(cast)
		case uint16:
			value = uint64(cast)
		case uint32:
			value = uint64(cast)
		case uint64:
			value = uint64(cast)
		case uintptr:
			value = uint64(cast)
		}

		n, err := writeVarUint(p.writer, value)
		total += n
		return total, err

	case reflect.Float32:
		binary.BigEndian.PutUint32(buffer[0:4], math.Float32bits(val.Interface().(float32)))

		n, err := p.writer.Write(buffer[0:4])
		total += n
		return total, err

	case reflect.Float64:
		binary.BigEndian.PutUint64(buffer[0:8], math.Float64bits(val.Interface().(float64)))

		n, err := p.writer.Write(buffer[0:8])
		total += n
		return total, err

	case reflect.Complex64:
		complex := val.Interface().(complex64)
		binary.BigEndian.PutUint32(buffer[0:4], math.Float32bits(real(complex)))
		binary.BigEndian.PutUint32(buffer[4:8], math.Float32bits(imag(complex)))

		n, err := p.writer.Write(buffer[:8])
		total += n
		return total, err

	case reflect.Complex128:
		complex := val.Interface().(complex128)
		binary.BigEndian.PutUint64(buffer[0:8], math.Float64bits(real(complex)))
		n, err := p.writer.Write(buffer[:8])
		total += n
		if err != nil {
			return total, err
		}

		binary.BigEndian.PutUint64(buffer[0:8], math.Float64bits(imag(complex)))
		n, err = p.writer.Write(buffer[:8])
		total += n
		if err != nil {
			return total, err
		}

		return total, err

	case reflect.Array:
		var (
			isInterface = typ.Elem().Kind() == reflect.Interface
			ln          = typ.Len()
		)

		for i := 0; i < ln; i++ {
			n, err := p.encode(val.Index(i).Interface(), packerInfo{markType: isInterface})
			total += n
			if err != nil {
				return total, err
			}
		}

		return total, nil

	case reflect.Map:
		var (
			isInterface = typ.Elem().Kind() == reflect.Interface
			ln          = val.Len()
		)

		if info.maxSize > 0 && uint64(ln) > info.maxSize {
			return total, &ErrDataTooLarge{typ: reflect.TypeOf(data), max: info.maxSize, size: uint64(ln)}
		}

		// If map is nil, set length to -1 so the decoder knows not to instance it
		if val.IsNil() {
			ln = -1
		}

		n, err := writeVarInt(p.writer, int64(ln))
		total += n
		if err != nil {
			return total, err
		}

		if ln <= 0 {
			return total, nil
		}

		iter := val.MapRange()

		for iter.Next() {
			var (
				curKey = iter.Key()
				curVal = iter.Value()
			)

			n, err := p.encode(curKey.Interface(), packerInfo{})
			total += n
			if err != nil {
				return total, err
			}

			n, err = p.encode(curVal.Interface(), packerInfo{markType: isInterface})
			total += n
			if err != nil {
				return total, err
			}
		}

		return total, nil

	case reflect.Slice:
		switch typ.Elem().Kind() {
		case reflect.Uint8:
			n, err := p.encodeBytes(val.Bytes(), info)
			total += n
			if err != nil {
				return total, err
			}
			return total, nil
		}

		var (
			isInterface = typ.Elem().Kind() == reflect.Interface
			ln          = val.Len()
		)

		if info.maxSize > 0 && uint64(ln) > info.maxSize {
			return total, &ErrDataTooLarge{typ: typ, max: info.maxSize, size: uint64(ln)}
		}

		n, err := writeVarUint(p.writer, uint64(ln))
		total += n
		if err != nil {
			return total, err
		}

		for i := 0; i < ln; i++ {
			n, err := p.encode(val.Index(i).Interface(), packerInfo{markType: isInterface})
			total += n
			if err != nil {
				return total, err
			}
		}

		return total, err

	case reflect.String:
		var encoded = []byte(val.Interface().(string))

		n, err := p.encodeBytes(encoded, info)
		total += n
		if err != nil {
			return total, err
		}
		return total, nil

	case reflect.Struct:
		ln := typ.NumField()

		for i := 0; i < ln; i++ {
			var curFieldTyp = typ.Field(i)

			if !curFieldTyp.IsExported() {
				continue
			}

			var (
				curVal = val.Field(i)

				curInfo = parsePackerInfo(curFieldTyp.Tag.Get("pack"))

				isInterface = curFieldTyp.Type.Kind() == reflect.Interface
			)

			curInfo.markType = isInterface

			n, err := p.encode(curVal.Interface(), curInfo)
			total += n
			if err != nil {
				return total, err
			}
		}

		return total, nil
	}

	return total, &ErrInvalidType{typ}
}
