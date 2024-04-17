package pack

import (
	"encoding/binary"
	"io"
	"math"
	"reflect"
	"unsafe"
)

type Packer interface {
	// Encode object into stream
	//
	// Note: If a pointer is given to the encoder (*obj), then the decoder needs to receive a pointer to a pointer (**obj)
	// (unless object mode is enabled in the options)
	Encode(data any) error

	// Total bytes written to the underlying stream
	BytesWritten() uint64

	// Reset bytes written to 0
	ResetCounter()

	// Set objects
	SetObjects(objects Objects)

	// Set sub-objects
	SetSubObjects(subObjects map[string]Objects)

	// Set size limit
	SetSizeLimit(sizeLimit uint64)
}

type packer struct {
	realWriter io.Writer

	writer  io.Writer
	written uint64
	buffer  dataBuffer

	objects   Objects
	subobj    map[string]Objects
	sizelimit uint64
	stopat    uint64
}

func NewPacker(writer io.Writer, options ...Options) Packer {
	p := &packer{realWriter: writer, subobj: map[string]Objects{}}

	for _, opt := range options {
		if opt.WithObjects != nil {
			p.objects = opt.WithObjects
		}
		if opt.SizeLimit > 0 {
			p.sizelimit = opt.SizeLimit
		}
		for key, opt := range opt.WithSubObjects {
			p.subobj[key] = opt
		}
	}

	if p.sizelimit <= 0 {
		p.writer = p.realWriter
	}

	return p
}

func (p *packer) Encode(data any) error {
	if p.sizelimit > 0 {
		p.stopat = p.written + p.sizelimit
		p.writer = &limitedWriter{
			O: p.sizelimit,
			N: p.sizelimit,
			W: p.realWriter,
		}
	}

	if p.objects != nil {
		return p.encodeObject(data, p.objects, packerInfo{})
	}

	return p.encode(data, packerInfo{})
}

func (p *packer) BytesWritten() uint64 {
	return p.written
}

func (p *packer) ResetCounter() {
	p.written = 0
}

func (p *packer) SetObjects(objects Objects) {
	p.objects = objects
}

func (p *packer) SetSubObjects(subObjects map[string]Objects) {
	p.subobj = subObjects
}

func (p *packer) SetSizeLimit(sizeLimit uint64) {
	p.sizelimit = sizeLimit
}

func (p *packer) encodeObject(data any, objects Objects, info packerInfo) error {
	for reflect.TypeOf(data).Kind() == reflect.Pointer {
		val := reflect.ValueOf(data)
		if val.IsNil() {
			return ErrNilObject
		}
		data = val.Elem().Interface()
	}

	oid, exists := objects.GetID(data)
	if !exists {
		return &ErrNotDefined{typ: reflect.TypeOf(data)}
	}

	n, err := writeVarUint(p.writer, uint64(oid), p.buffer[:])
	p.written += uint64(n)
	if err != nil {
		return err
	}

	info.forceAsObject = true

	return p.encode(data, info)
}

func (p *packer) encodeBytes(data []byte, inf packerInfo) error {
	var ln = uint64(len(data))

	if inf.maxSize > 0 && ln > inf.maxSize {
		return &ErrDataTooLarge{typ: reflect.TypeOf(data), max: inf.maxSize, size: uint64(len(data))}
	}

	if p.stopat > 0 && p.written+ln > p.stopat {
		return &ErrDataTooLarge{max: p.sizelimit, size: p.written + ln}
	}

	n, err := writeVarUint(p.writer, ln, p.buffer[:])
	p.written += uint64(n)
	if err != nil {
		return err
	}

	n, err = p.writer.Write(data)
	p.written += uint64(n)

	return err
}

func (p *packer) encodeBoolSlice(data []bool, inf packerInfo) error {
	var ln = uint64((len(data) + 7) / 8)

	if inf.maxSize > 0 && ln > inf.maxSize {
		return &ErrDataTooLarge{typ: reflect.TypeOf(data), max: inf.maxSize, size: uint64(len(data))}
	}

	if p.stopat > 0 && p.written+ln > p.stopat {
		return &ErrDataTooLarge{max: p.sizelimit, size: p.written + ln}
	}

	n, err := writeVarUint(p.writer, uint64(len(data)), p.buffer[:])
	p.written += uint64(n)
	if err != nil {
		return err
	}

	for j := 0; j < len(data); j += 8 {
		p.buffer[0] = 0
		for i := 0; i < 8 && j+i < len(data); i++ {
			if data[j+i] {
				p.buffer[0] |= byte(1 << i)
			}
		}
		n, err = p.writer.Write(p.buffer[:1])
		p.written += uint64(n)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *packer) encodeType(typ reflect.Type) error {

	var kind reflect.Kind

	if typ == nil {
		kind = 0xff
	} else {
		kind = typ.Kind()
	}

	if !canEncodeInInterface[kind] {
		return &ErrCantUseInInterfaceMode{kind: kind, typ: typ}
	}

	p.buffer[0] = byte(kind)
	n, err := p.writer.Write(p.buffer[:1])
	p.written += uint64(n)
	if err != nil {
		return err
	}

	switch kind {
	case reflect.Pointer:
		err = p.encodeType(typ.Elem())
		if err != nil {
			return err
		}

	case reflect.Array:
		n, err := writeVarUint(p.writer, uint64(typ.Len()), p.buffer[:])
		p.written += uint64(n)
		if err != nil {
			return err
		}

		err = p.encodeType(typ.Elem())
		if err != nil {
			return err
		}

	case reflect.Map:
		err = p.encodeType(typ.Key())
		if err != nil {
			return err
		}

		err = p.encodeType(typ.Elem())
		if err != nil {
			return err
		}

	case reflect.Slice:
		err = p.encodeType(typ.Elem())
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *packer) encode(data any, info packerInfo) error {
	if info.ignore {
		return nil
	}

	var (
		n   int
		err error

		typ = reflect.TypeOf(data)
		val = reflect.ValueOf(data)
	)

	if info.markType {
		err = p.encodeType(typ)
		if err != nil {
			return err
		}

		if typ == nil {
			return nil
		}
	}

	if typ == nil {
		return ErrNil
	}

	for typ.Kind() == reflect.Pointer {
		if val.IsNil() {
			p.buffer[0] = 0
			n, err = p.writer.Write(p.buffer[:1])
			p.written += uint64(n)
			return err
		}

		p.buffer[0] = 1
		n, err = p.writer.Write(p.buffer[:1])
		p.written += uint64(n)
		if err != nil {
			return err
		}

		typ = typ.Elem()
		val = val.Elem()
	}

	switch typ.Kind() {
	case reflect.Bool:
		if val.Bool() {
			p.buffer[0] = 1
		} else {
			p.buffer[0] = 0
		}
		n, err = p.writer.Write(p.buffer[:1])
		p.written += uint64(n)
		return err

	case reflect.Int8:
		p.buffer[0] = byte(val.Int())
		n, err = p.writer.Write(p.buffer[:1])
		p.written += uint64(n)
		return err

	case reflect.Uint8:
		p.buffer[0] = byte(val.Uint())
		n, err = p.writer.Write(p.buffer[:1])
		p.written += uint64(n)
		return err

	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err = writeVarInt(p.writer, val.Int(), p.buffer[:])
		p.written += uint64(n)
		return err

	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		n, err = writeVarUint(p.writer, val.Uint(), p.buffer[:])
		p.written += uint64(n)
		return err

	case reflect.Float32:
		binary.BigEndian.PutUint32(p.buffer[0:4], math.Float32bits(val.Interface().(float32)))

		n, err = p.writer.Write(p.buffer[0:4])
		p.written += uint64(n)
		return err

	case reflect.Float64:
		binary.BigEndian.PutUint64(p.buffer[0:8], math.Float64bits(val.Interface().(float64)))

		n, err = p.writer.Write(p.buffer[0:8])
		p.written += uint64(n)
		return err

	case reflect.Complex64:
		complex := val.Interface().(complex64)
		binary.BigEndian.PutUint32(p.buffer[0:4], math.Float32bits(real(complex)))
		binary.BigEndian.PutUint32(p.buffer[4:8], math.Float32bits(imag(complex)))

		n, err = p.writer.Write(p.buffer[:8])
		p.written += uint64(n)
		return err

	case reflect.Complex128:
		complex := val.Interface().(complex128)
		binary.BigEndian.PutUint64(p.buffer[0:8], math.Float64bits(real(complex)))
		n, err = p.writer.Write(p.buffer[:8])
		p.written += uint64(n)
		if err != nil {
			return err
		}

		binary.BigEndian.PutUint64(p.buffer[0:8], math.Float64bits(imag(complex)))
		n, err = p.writer.Write(p.buffer[:8])
		p.written += uint64(n)
		return err

	case reflect.Array:
		var (
			isInterface = typ.Elem().Kind() == reflect.Interface
			ln          = typ.Len()
		)

		if objects, ok := p.subobj[info.objects]; ok {
			for i := 0; i < ln; i++ {
				err = p.encodeObject(val.Index(i).Interface(), objects, packerInfo{})
				if err != nil {
					return err
				}
			}
		} else {
			for i := 0; i < ln; i++ {
				err = p.encode(val.Index(i).Interface(), packerInfo{markType: isInterface})
				if err != nil {
					return err
				}
			}
		}

		return nil

	case reflect.Map:
		var (
			isInterface = typ.Elem().Kind() == reflect.Interface
			ln          = val.Len()
		)

		if info.maxSize > 0 && uint64(ln) > info.maxSize {
			return &ErrDataTooLarge{typ: reflect.TypeOf(data), max: info.maxSize, size: uint64(ln)}
		}

		if p.stopat > 0 && p.written+uint64(ln) > p.stopat {
			return &ErrDataTooLarge{max: p.sizelimit, size: p.written + uint64(ln)}
		}

		// If map is nil, set length to -1 so the decoder knows not to instance it
		if val.IsNil() {
			ln = -1
		}

		n, err = writeVarInt(p.writer, int64(ln), p.buffer[:])
		p.written += uint64(n)
		if err != nil {
			return err
		}

		if ln <= 0 {
			return nil
		}

		iter := val.MapRange()

		if objects, ok := p.subobj[info.objects]; ok {
			for i := 0; i < ln; i++ {
			}
			for iter.Next() {
				var (
					curKey = iter.Key()
					curVal = iter.Value()
				)

				err = p.encode(curKey.Interface(), packerInfo{})
				if err != nil {
					return err
				}

				err = p.encodeObject(curVal.Interface(), objects, packerInfo{})
				if err != nil {
					return err
				}
			}
		} else {
			for iter.Next() {
				var (
					curKey = iter.Key()
					curVal = iter.Value()
				)

				err = p.encode(curKey.Interface(), packerInfo{})
				if err != nil {
					return err
				}

				err = p.encode(curVal.Interface(), packerInfo{markType: isInterface})
				if err != nil {
					return err
				}
			}
		}

		return nil

	case reflect.Slice:

		switch typ.Elem().Kind() {
		case reflect.Uint8:
			return p.encodeBytes(val.Bytes(), info)

		case reflect.Bool:
			return p.encodeBoolSlice(val.Interface().([]bool), info)
		}

		var (
			isInterface = typ.Elem().Kind() == reflect.Interface
			ln          = val.Len()
		)

		if info.maxSize > 0 && uint64(ln) > info.maxSize {
			return &ErrDataTooLarge{typ: typ, max: info.maxSize, size: uint64(ln)}
		}

		if p.stopat > 0 && p.written+uint64(ln) > p.stopat {
			return &ErrDataTooLarge{max: p.sizelimit, size: p.written + uint64(ln)}
		}

		n, err = writeVarUint(p.writer, uint64(ln), p.buffer[:])
		p.written += uint64(n)
		if err != nil {
			return err
		}

		if objects, ok := p.subobj[info.objects]; ok {
			for i := 0; i < ln; i++ {
				err = p.encodeObject(val.Index(i).Interface(), objects, packerInfo{})
				if err != nil {
					return err
				}
			}
		} else {
			for i := 0; i < ln; i++ {
				err := p.encode(val.Index(i).Interface(), packerInfo{markType: isInterface})
				if err != nil {
					return err
				}
			}
		}

		return err

	case reflect.String:
		var (
			str     = val.Interface().(string)
			encoded = *(*[]byte)(unsafe.Pointer(&str))
		)

		return p.encodeBytes(encoded, info)

	case reflect.Struct:
		if val.CanAddr() && reflect.PointerTo(typ).Implements(interfaceBeforePack) {
			err := val.Addr().Interface().(BeforePack).BeforePack()
			if err != nil {
				return err
			}
		} else if typ.Implements(interfaceBeforePack) {
			err := val.Interface().(BeforePack).BeforePack()
			if err != nil {
				return err
			}
		}

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

			if objects, ok := p.subobj[curInfo.objects]; ok && isInterface {
				err := p.encodeObject(curVal.Interface(), objects, curInfo)
				if err != nil {
					return err
				}
			} else {
				curInfo.markType = isInterface
				err := p.encode(curVal.Interface(), curInfo)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}

	return &ErrInvalidType{typ}
}
