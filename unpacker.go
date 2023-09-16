package pack

import (
	"encoding/binary"
	"io"
	"math"
	"reflect"
)

var typePointerToInterface = reflect.PointerTo(reflect.TypeOf([]any{}).Elem())

type Unpacker interface {
	Decode(data any) error

	BytesRead() uint64

	WithObjects(Objects) Unpacker
}

type unpacker struct {
	reader   io.Reader
	maxalloc uint64
	read     uint64

	objects Objects
}

func NewUnpacker(reader io.Reader) Unpacker {
	return &unpacker{reader: reader, maxalloc: 0xffff_ffff}
}

func (u *unpacker) Decode(data any) error {

	if u.objects != nil {

		if reflect.TypeOf(data) != typePointerToInterface {
			return ErrMustBePointerToInterface
		}

		var oid uint64

		n, err := readVarUint(u.reader, &oid)
		u.read += uint64(n)
		if err != nil {
			return err
		}

		typ, exists := u.objects.GetType(uint(oid))
		if !exists {
			return &ErrNotDefined{oid: uint(oid)}
		}

		item := reflect.New(typ)

		read, err := u.decode(item.Interface(), packerInfo{})
		u.read += uint64(read)
		if err != nil {
			return err
		}

		reflect.ValueOf(data).Elem().Set(item)

		return nil

	}

	read, err := u.decode(data, packerInfo{})
	u.read += uint64(read)

	return err
}

func (u *unpacker) BytesRead() uint64 {
	return u.read
}

func (u *unpacker) WithObjects(o Objects) Unpacker {
	u.objects = o
	return u
}

func (u *unpacker) decodeBytes(ln uint64, info packerInfo) ([]byte, int, error) {
	if info.maxSize > 0 && ln > uint64(info.maxSize) {
		return nil, 0, &ErrDataTooLarge{typ: reflect.TypeOf([]byte{}), max: info.maxSize, size: ln}
	}

	if ln > u.maxalloc {
		return nil, 0, &ErrMaxAlloc{request: ln, allowed: u.maxalloc}
	}

	var buf = make([]byte, int(ln))

	n, err := io.ReadFull(u.reader, buf)
	if err != nil {
		return nil, n, err
	}

	return buf, n, err
}

func (u *unpacker) decodeType() (reflect.Type, int, error) {
	var (
		total int
		buf   [1]byte
	)

	n, err := u.reader.Read(buf[:1])
	total += n
	if err != nil {
		return nil, total, err
	}

	var kind = reflect.Kind(buf[0])

	switch kind {
	case reflect.Array:
		var ln uint64

		n, err := readVarUint(u.reader, &ln)
		total += n
		if err != nil {
			return nil, total, err
		}

		innerType, n, err := u.decodeType()
		total += n
		if err != nil {
			return nil, total, err
		}

		if innerType == nil {
			return nil, total, ErrNil
		}

		return reflect.ArrayOf(int(ln), innerType), total, nil

	case reflect.Map:
		keyType, n, err := u.decodeType()
		total += n
		if err != nil {
			return nil, total, err
		}

		if keyType == nil {
			return nil, total, ErrNil
		}

		valType, n, err := u.decodeType()
		total += n
		if err != nil {
			return nil, total, err
		}

		if valType == nil {
			return nil, total, ErrNil
		}

		return reflect.MapOf(keyType, valType), total, nil

	case reflect.Slice:
		innerType, n, err := u.decodeType()
		total += n
		if err != nil {
			return nil, total, err
		}

		if innerType == nil {
			return nil, total, ErrNil
		}

		return reflect.SliceOf(innerType), total, nil

	case reflect.Pointer:
		innerType, n, err := u.decodeType()
		total += n
		if err != nil {
			return nil, total, err
		}

		if innerType == nil {
			return nil, total, ErrNil
		}

		return reflect.PointerTo(innerType), total, nil

	case reflect.Interface:
		return reflect.TypeOf([]any{}).Elem(), total, nil

	case reflect.Bool:
		return reflect.TypeOf(false), total, nil

	case reflect.Int:
		return reflect.TypeOf(int(0)), total, nil

	case reflect.Int8:
		return reflect.TypeOf(int8(0)), total, nil

	case reflect.Int16:
		return reflect.TypeOf(int16(0)), total, nil

	case reflect.Int32:
		return reflect.TypeOf(int32(0)), total, nil

	case reflect.Int64:
		return reflect.TypeOf(int64(0)), total, nil

	case reflect.Uint:
		return reflect.TypeOf(uint(0)), total, nil

	case reflect.Uint8:
		return reflect.TypeOf(uint8(0)), total, nil

	case reflect.Uint16:
		return reflect.TypeOf(uint16(0)), total, nil

	case reflect.Uint32:
		return reflect.TypeOf(uint32(0)), total, nil

	case reflect.Uint64:
		return reflect.TypeOf(uint64(0)), total, nil

	case reflect.Uintptr:
		return reflect.TypeOf(uintptr(0)), total, nil

	case reflect.Float32:
		return reflect.TypeOf(float32(0)), total, nil

	case reflect.Float64:
		return reflect.TypeOf(float64(0)), total, nil

	case reflect.Complex64:
		return reflect.TypeOf(complex64(complex(0, 0))), total, nil

	case reflect.Complex128:
		return reflect.TypeOf(complex128(complex(0, 0))), total, nil

	case reflect.String:
		return reflect.TypeOf(""), total, nil

	case 0xff:
		return nil, total, nil

	}

	return nil, total, ErrInvalidReceiver
}

func (u *unpacker) decodeMarked(info packerInfo) (reflect.Value, int, error) {
	var (
		total    int
		receiver reflect.Value
	)

	typ, n, err := u.decodeType()
	total += n
	if err != nil {
		return reflect.Value{}, total, err
	}

	if typ == nil {
		return reflect.Value{}, total, nil
	}

	receiver = reflect.New(typ)

	n, err = u.decode(receiver.Interface(), info)
	total += n
	if err != nil {
		return reflect.Value{}, total, err
	}

	return receiver.Elem(), total, err
}

func (u *unpacker) decode(data any, info packerInfo) (int, error) {
	if info.ignore {
		return 0, nil
	}

	if reflect.TypeOf(data).Kind() != reflect.Pointer {
		return 0, ErrInvalidReceiver
	}

	var (
		typ = reflect.TypeOf(data).Elem()
		val = reflect.ValueOf(data).Elem()

		total int

		buf [8]byte
	)

	switch typ.Kind() {
	case reflect.Pointer:
		n, err := u.reader.Read(buf[:1])
		total += n
		if err != nil {
			return total, err
		}

		if buf[0] == 0 {
			return total, nil
		}

		item := reflect.New(typ.Elem())

		n, err = u.decode(item.Interface(), packerInfo{})
		total += n
		if err != nil {
			return total, err
		}

		val.Set(item)

		return total, nil

	case reflect.Bool, reflect.Int8, reflect.Uint8:
		n, err := u.reader.Read(buf[:1])
		total += n
		if err != nil {
			return total, err
		}

		switch typ.Kind() {

		case reflect.Bool:
			if buf[0] == 0 {
				val.SetBool(false)
			} else {
				val.SetBool(true)
			}

		case reflect.Int8:
			val.SetInt(int64(buf[0]))

		case reflect.Uint8:
			val.SetUint(uint64(buf[0]))

		}

		return total, nil

	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
		var num int64

		n, err := readVarInt(u.reader, &num)
		total += n
		if err != nil {
			return total, err
		}

		val.SetInt(num)

		return total, nil

	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		var num uint64

		n, err := readVarUint(u.reader, &num)
		total += n
		if err != nil {
			return total, err
		}

		val.SetUint(num)

		return total, nil

	case reflect.Float32:
		n, err := io.ReadFull(u.reader, buf[0:4])
		total += n
		if err != nil {
			return total, err
		}

		val.SetFloat(float64(math.Float32frombits(binary.BigEndian.Uint32(buf[0:4]))))

		return total, nil

	case reflect.Float64:
		n, err := io.ReadFull(u.reader, buf[0:8])
		total += n
		if err != nil {
			return total, err
		}

		val.SetFloat(math.Float64frombits(binary.BigEndian.Uint64(buf[0:8])))

		return total, nil

	case reflect.Complex64:
		n, err := io.ReadFull(u.reader, buf[0:8])
		total += n
		if err != nil {
			return total, err
		}

		var (
			r = float64(math.Float32frombits(binary.BigEndian.Uint32(buf[0:4])))
			i = float64(math.Float32frombits(binary.BigEndian.Uint32(buf[4:8])))
		)

		val.SetComplex(complex(r, i))

		return total, nil

	case reflect.Complex128:
		n, err := io.ReadFull(u.reader, buf[0:8])
		total += n
		if err != nil {
			return total, err
		}

		var r = math.Float64frombits(binary.BigEndian.Uint64(buf[0:8]))

		n, err = io.ReadFull(u.reader, buf[0:8])
		total += n
		if err != nil {
			return total, err
		}

		var i = math.Float64frombits(binary.BigEndian.Uint64(buf[0:8]))

		val.SetComplex(complex(r, i))

		return total, nil

	case reflect.Array:

		var (
			isInterface = typ.Elem().Kind() == reflect.Interface
			ln          = typ.Len()
		)

		if isInterface {
			for i := 0; i < ln; i++ {
				elem, n, err := u.decodeMarked(packerInfo{})
				total += n
				if err != nil {
					return total, err
				}
				val.Index(i).Set(elem)
			}
		} else {
			for i := 0; i < ln; i++ {
				n, err := u.decode(val.Index(i).Addr().Interface(), packerInfo{})
				total += n
				if err != nil {
					return total, err
				}
			}
		}

		return total, nil

	case reflect.Map:
		var (
			isInterface = typ.Elem().Kind() == reflect.Interface
			ln          int64
		)

		n, err := readVarInt(u.reader, &ln)
		total += n
		if err != nil {
			return total, err
		}

		// If length is negative, keep map as nil
		if ln < 0 {
			return total, nil
		}

		val.Set(reflect.MakeMap(typ))

		for i := 0; i < int(ln); i++ {
			curKey := reflect.New(typ.Key())

			n, err := u.decode(curKey.Interface(), packerInfo{})
			total += n
			if err != nil {
				return total, err
			}

			var curVal reflect.Value

			if isInterface {
				curVal, n, err = u.decodeMarked(packerInfo{})
				total += n
				if err != nil {
					return total, err
				}
			} else {
				curVal = reflect.New(typ.Elem())

				n, err := u.decode(curVal.Interface(), packerInfo{})
				total += n
				if err != nil {
					return total, err
				}

				curVal = curVal.Elem()
			}

			val.SetMapIndex(curKey.Elem(), curVal)
		}

		return total, nil

	case reflect.Slice:

		var ln uint64

		n, err := readVarUint(u.reader, &ln)
		total += n
		if err != nil {
			return total, err
		}

		switch typ.Elem().Kind() {
		case reflect.Uint8:

			data, n, err := u.decodeBytes(ln, info)
			total += n
			if err != nil {
				return total, err
			}

			val.SetBytes(data)

			return total, nil
		}

		var isInterface = typ.Elem().Kind() == reflect.Interface

		val.Set(reflect.MakeSlice(typ, int(ln), int(ln)))

		if isInterface {
			for i := 0; i < int(ln); i++ {
				curItem, n, err := u.decodeMarked(packerInfo{})
				total += n
				if err != nil {
					return total, err
				}

				val.Index(i).Set(curItem)
			}
		} else {
			for i := 0; i < int(ln); i++ {
				n, err := u.decode(val.Index(i).Addr().Interface(), packerInfo{})
				total += n
				if err != nil {
					return total, err
				}
			}
		}

		return total, nil

	case reflect.String:
		var ln uint64

		n, err := readVarUint(u.reader, &ln)
		total += n
		if err != nil {
			return total, err
		}

		buf, n, err := u.decodeBytes(ln, info)
		total += n
		if err != nil {
			return total, err
		}

		val.SetString(string(buf))
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

			if isInterface {
				item, n, err := u.decodeMarked(curInfo)
				total += n
				if err != nil {
					return total, err
				}

				if (item != reflect.Value{}) {
					curVal.Set(item)
				}
			} else {
				n, err := u.decode(curVal.Addr().Interface(), curInfo)
				total += n
				if err != nil {
					return total, err
				}
			}
		}

		return total, nil

	}

	return total, &ErrInvalidType{typ}
}
