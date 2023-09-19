package pack

import (
	"encoding/binary"
	"io"
	"math"
	"reflect"
	"unsafe"
)

var typePointerToInterface = reflect.PointerTo(reflect.TypeOf([]any{}).Elem())

type Unpacker interface {
	// Decode object on stream into a pointer
	Decode(data any) error

	// Total bytes read from the underlying stream
	BytesRead() uint64
}

type unpacker struct {
	realReader io.Reader

	reader io.Reader
	read   uint64
	buffer dataBuffer

	objects   Objects
	sizelimit uint64
	stopat    uint64
}

func NewUnpacker(reader io.Reader, options ...Options) Unpacker {
	u := &unpacker{realReader: reader}

	for _, opt := range options {
		if opt.WithObjects != nil {
			u.objects = opt.WithObjects
		}
		if opt.SizeLimit > 0 {
			u.sizelimit = opt.SizeLimit
		}
	}

	if u.sizelimit <= 0 {
		u.reader = u.realReader
	}

	return u
}

func (u *unpacker) Decode(data any) error {

	if u.sizelimit > 0 {
		u.stopat = u.read + u.sizelimit
		u.reader = &limitedReader{
			O: u.sizelimit,
			N: u.sizelimit,
			R: u.realReader,
		}
	}

	if u.objects != nil {

		if reflect.TypeOf(data) != typePointerToInterface {
			return ErrMustBePointerToInterface
		}

		var oid uint64

		n, err := readVarUint(u.reader, &oid, u.buffer[:])
		u.read += uint64(n)
		if err != nil {
			return err
		}

		typ, exists := u.objects.GetType(uint(oid))
		if !exists {
			return &ErrNotDefined{oid: uint(oid)}
		}

		item := reflect.New(typ)

		err = u.decode(item.Interface(), packerInfo{})
		if err != nil {
			return err
		}

		reflect.ValueOf(data).Elem().Set(item)

		return nil

	}

	return u.decode(data, packerInfo{})
}

func (u *unpacker) BytesRead() uint64 {
	return u.read
}

func (u *unpacker) decodeBytes(ln uint64, info packerInfo) ([]byte, error) {
	if info.maxSize > 0 && ln > uint64(info.maxSize) {
		return nil, &ErrDataTooLarge{typ: reflect.TypeOf([]byte{}), max: info.maxSize, size: ln}
	}

	if u.stopat > 0 && u.read+ln > u.stopat {
		return nil, &ErrDataTooLarge{max: u.sizelimit, size: u.read + ln - (u.stopat - u.sizelimit)}
	}

	if ln == 0 {
		return nil, nil
	}

	var buf = make([]byte, int(ln))

	n, err := io.ReadFull(u.reader, buf)
	u.read += uint64(n)
	if err != nil {
		return nil, err
	}

	return buf, err
}

func (u *unpacker) decodeType() (reflect.Type, error) {
	n, err := u.reader.Read(u.buffer[:1])
	u.read += uint64(n)
	if err != nil {
		return nil, err
	}

	var kind = reflect.Kind(u.buffer[0])

	switch kind {
	case reflect.Array:
		var ln uint64

		n, err := readVarUint(u.reader, &ln, u.buffer[:])
		u.read += uint64(n)
		if err != nil {
			return nil, err
		}

		if u.stopat > 0 && u.read+ln > u.stopat {
			return nil, &ErrDataTooLarge{max: u.sizelimit, size: u.read + ln - (u.stopat - u.sizelimit)}
		}

		innerType, err := u.decodeType()
		if err != nil {
			return nil, err
		}

		if innerType == nil {
			return nil, ErrNil
		}

		return reflect.ArrayOf(int(ln), innerType), nil

	case reflect.Map:
		keyType, err := u.decodeType()
		if err != nil {
			return nil, err
		}

		if keyType == nil {
			return nil, ErrNil
		}

		valType, err := u.decodeType()
		if err != nil {
			return nil, err
		}

		if valType == nil {
			return nil, ErrNil
		}

		return reflect.MapOf(keyType, valType), nil

	case reflect.Slice:
		innerType, err := u.decodeType()
		if err != nil {
			return nil, err
		}

		if innerType == nil {
			return nil, ErrNil
		}

		return reflect.SliceOf(innerType), nil

	case reflect.Pointer:
		innerType, err := u.decodeType()
		if err != nil {
			return nil, err
		}

		if innerType == nil {
			return nil, ErrNil
		}

		return reflect.PointerTo(innerType), nil

	default:
		typ, ok := kindToType[kind]
		if ok {
			return typ, nil
		}

	}

	return nil, ErrInvalidReceiver
}

func (u *unpacker) decodeMarked(info packerInfo) (reflect.Value, error) {
	var (
		receiver reflect.Value
	)

	typ, err := u.decodeType()
	if err != nil {
		return reflect.Value{}, err
	}

	if typ == nil {
		return reflect.Value{}, nil
	}

	receiver = reflect.New(typ)

	err = u.decode(receiver.Interface(), info)
	if err != nil {
		return reflect.Value{}, err
	}

	return receiver.Elem(), err
}

func (u *unpacker) decode(data any, info packerInfo) error {
	if info.ignore {
		return nil
	}

	if reflect.TypeOf(data).Kind() != reflect.Pointer {
		return ErrInvalidReceiver
	}

	var (
		typ = reflect.TypeOf(data).Elem()
		val = reflect.ValueOf(data).Elem()
	)

	switch typ.Kind() {
	case reflect.Pointer:
		n, err := u.reader.Read(u.buffer[:1])
		u.read += uint64(n)
		if err != nil {
			return err
		}

		if u.buffer[0] == 0 {
			return nil
		}

		item := reflect.New(typ.Elem())

		err = u.decode(item.Interface(), packerInfo{})
		if err != nil {
			return err
		}

		val.Set(item)

		return nil

	case reflect.Bool, reflect.Int8, reflect.Uint8:
		n, err := u.reader.Read(u.buffer[:1])
		u.read += uint64(n)
		if err != nil {
			return err
		}

		switch typ.Kind() {

		case reflect.Bool:
			if u.buffer[0] == 0 {
				val.SetBool(false)
			} else {
				val.SetBool(true)
			}

		case reflect.Int8:
			val.SetInt(int64(u.buffer[0]))

		case reflect.Uint8:
			val.SetUint(uint64(u.buffer[0]))

		}

		return nil

	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
		var num int64

		n, err := readVarInt(u.reader, &num, u.buffer[:])
		u.read += uint64(n)
		if err != nil {
			return err
		}

		val.SetInt(num)

		return nil

	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		var num uint64

		n, err := readVarUint(u.reader, &num, u.buffer[:])
		u.read += uint64(n)
		if err != nil {
			return err
		}

		val.SetUint(num)

		return nil

	case reflect.Float32:
		n, err := io.ReadFull(u.reader, u.buffer[0:4])
		u.read += uint64(n)
		if err != nil {
			return err
		}

		val.SetFloat(float64(math.Float32frombits(binary.BigEndian.Uint32(u.buffer[0:4]))))

		return nil

	case reflect.Float64:
		n, err := io.ReadFull(u.reader, u.buffer[0:8])
		u.read += uint64(n)
		if err != nil {
			return err
		}

		val.SetFloat(math.Float64frombits(binary.BigEndian.Uint64(u.buffer[0:8])))

		return nil

	case reflect.Complex64:
		n, err := io.ReadFull(u.reader, u.buffer[0:8])
		u.read += uint64(n)
		if err != nil {
			return err
		}

		var (
			r = float64(math.Float32frombits(binary.BigEndian.Uint32(u.buffer[0:4])))
			i = float64(math.Float32frombits(binary.BigEndian.Uint32(u.buffer[4:8])))
		)

		val.SetComplex(complex(r, i))

		return nil

	case reflect.Complex128:
		n, err := io.ReadFull(u.reader, u.buffer[0:8])
		u.read += uint64(n)
		if err != nil {
			return err
		}

		var r = math.Float64frombits(binary.BigEndian.Uint64(u.buffer[0:8]))

		n, err = io.ReadFull(u.reader, u.buffer[0:8])
		u.read += uint64(n)
		if err != nil {
			return err
		}

		var i = math.Float64frombits(binary.BigEndian.Uint64(u.buffer[0:8]))

		val.SetComplex(complex(r, i))

		return nil

	case reflect.Array:

		var (
			isInterface = typ.Elem().Kind() == reflect.Interface
			ln          = typ.Len()
		)

		if info.maxSize > 0 && uint64(ln) > info.maxSize {
			return &ErrDataTooLarge{typ: reflect.TypeOf(data), max: info.maxSize, size: uint64(ln)}
		}

		if u.stopat > 0 && u.read+uint64(ln) > u.stopat {
			return &ErrDataTooLarge{max: u.sizelimit, size: u.read + uint64(ln) - (u.stopat - u.sizelimit)}
		}

		if isInterface {
			for i := 0; i < ln; i++ {
				elem, err := u.decodeMarked(packerInfo{})
				if err != nil {
					return err
				}
				val.Index(i).Set(elem)
			}
		} else {
			for i := 0; i < ln; i++ {
				err := u.decode(val.Index(i).Addr().Interface(), packerInfo{})
				if err != nil {
					return err
				}
			}
		}

		return nil

	case reflect.Map:
		var (
			isInterface = typ.Elem().Kind() == reflect.Interface
			ln          int64
		)

		n, err := readVarInt(u.reader, &ln, u.buffer[:])
		u.read += uint64(n)
		if err != nil {
			return err
		}

		// If length is negative, set map to nil
		if ln < 0 {
			val.SetZero()
			return nil
		}

		if info.maxSize > 0 && uint64(ln) > info.maxSize {
			return &ErrDataTooLarge{typ: reflect.TypeOf(data), max: info.maxSize, size: uint64(ln)}
		}

		if u.stopat > 0 && u.read+uint64(ln) > u.stopat {
			return &ErrDataTooLarge{max: u.sizelimit, size: u.read + uint64(ln) - (u.stopat - u.sizelimit)}
		}

		val.Set(reflect.MakeMap(typ))

		for i := 0; i < int(ln); i++ {
			curKey := reflect.New(typ.Key())

			err := u.decode(curKey.Interface(), packerInfo{})
			if err != nil {
				return err
			}

			var curVal reflect.Value

			if isInterface {
				curVal, err = u.decodeMarked(packerInfo{})
				if err != nil {
					return err
				}
			} else {
				curVal = reflect.New(typ.Elem())

				err := u.decode(curVal.Interface(), packerInfo{})
				if err != nil {
					return err
				}

				curVal = curVal.Elem()
			}

			val.SetMapIndex(curKey.Elem(), curVal)
		}

		return nil

	case reflect.Slice:

		var ln uint64

		n, err := readVarUint(u.reader, &ln, u.buffer[:])
		u.read += uint64(n)
		if err != nil {
			return err
		}

		switch typ.Elem().Kind() {
		case reflect.Uint8:

			data, err := u.decodeBytes(ln, info)
			if err != nil {
				return err
			}

			val.SetBytes(data)

			return nil
		}

		if info.maxSize > 0 && uint64(ln) > info.maxSize {
			return &ErrDataTooLarge{typ: reflect.TypeOf(data), max: info.maxSize, size: uint64(ln)}
		}

		if u.stopat > 0 && u.read+ln > u.stopat {
			return &ErrDataTooLarge{max: u.sizelimit, size: u.read + ln - (u.stopat - u.sizelimit)}
		}

		var isInterface = typ.Elem().Kind() == reflect.Interface

		val.Set(reflect.MakeSlice(typ, int(ln), int(ln)))

		if isInterface {
			for i := 0; i < int(ln); i++ {
				curItem, err := u.decodeMarked(packerInfo{})
				if err != nil {
					return err
				}

				val.Index(i).Set(curItem)
			}
		} else {
			for i := 0; i < int(ln); i++ {
				err := u.decode(val.Index(i).Addr().Interface(), packerInfo{})
				if err != nil {
					return err
				}
			}
		}

		return nil

	case reflect.String:
		var ln uint64

		n, err := readVarUint(u.reader, &ln, u.buffer[:])
		u.read += uint64(n)
		if err != nil {
			return err
		}

		buf, err := u.decodeBytes(ln, info)
		if err != nil {
			return err
		}

		val.SetString(*(*string)(unsafe.Pointer(&buf)))

		return nil

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

			if isInterface {
				item, err := u.decodeMarked(curInfo)
				if err != nil {
					return err
				}

				if (item != reflect.Value{}) {
					curVal.Set(item)
				}
			} else {
				err := u.decode(curVal.Addr().Interface(), curInfo)
				if err != nil {
					return err
				}
			}
		}

		if val.CanAddr() && reflect.PointerTo(typ).Implements(interfaceAfterUnpack) {
			err := val.Addr().Interface().(AfterUnpack).AfterUnpack()
			if err != nil {
				return err
			}
		} else if typ.Implements(interfaceAfterUnpack) {
			err := val.Interface().(AfterUnpack).AfterUnpack()
			if err != nil {
				return err
			}
		}

		return nil

	}

	return &ErrInvalidType{typ}
}
