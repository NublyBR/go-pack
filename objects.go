package pack

import "reflect"

type Objects interface {
	GetID(item any) (uint, bool)

	GetType(id uint) (reflect.Type, bool)

	Push(items ...any) Objects
}

type objects struct {
	lastID   uint
	idToType map[uint]reflect.Type
	typeToId map[reflect.Type]uint
}

func NewObjects(items ...any) Objects {
	return (&objects{
		lastID:   0,
		idToType: make(map[uint]reflect.Type, len(items)),
		typeToId: make(map[reflect.Type]uint, len(items)),
	}).Push(items...)
}

func (o *objects) GetID(item any) (uint, bool) {
	typ := reflect.TypeOf(item)

	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	id, ok := o.typeToId[typ]

	return id, ok
}

func (o *objects) GetType(id uint) (reflect.Type, bool) {
	typ, ok := o.idToType[id]

	return typ, ok
}

func (o *objects) Push(items ...any) Objects {
	for _, item := range items {
		o.lastID += 1

		typ := reflect.TypeOf(item)

		for typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}

		o.idToType[o.lastID] = typ
		o.typeToId[typ] = o.lastID
	}

	return o
}
