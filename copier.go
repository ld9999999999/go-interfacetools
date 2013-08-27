// BSD licensed

package interfacetools

import (
	"errors"
	"reflect"
	"bytes"
	"fmt"
)

// Performs a deep-copy of source object to the passed in interface.
//
// If the object is a struct, then the copy is attempted based on the tag
// of the json tag of the field; the same rules apply per encoding/json.
// If a field is not available, then a copier skips it.
func CopyOut(src interface{}, out interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr {
		return errors.New("output interface must be a pointer")
	}

	dec := newDecoder()
	iv := reflect.ValueOf(src)
	ov := reflect.ValueOf(out)
	return dec.decode(iv, ov)
}

type decoder struct {
	path   []string
}

func newDecoder() *decoder {
	return &decoder{make([]string, 0, 12)}
}

func (d *decoder) pathString() string {
	if len(d.path) == 0 {
		return ""
	}

	var buf bytes.Buffer
	for i := range d.path {
		if i > 0 {
			buf.WriteString(".")
		}
		buf.WriteString(d.path[i])
	}
	buf.WriteString(" ")
	return buf.String()
}

func (d *decoder) pushPath(s string) {
	if d.path != nil {
		d.path = append(d.path, s)
	}
}

func (d *decoder) popPath() string {
	if len(d.path) > 0 {
		s := d.path[len(d.path)-1]
		d.path = d.path[:len(d.path)-1]
		return s
	}
	return ""
}

func (d *decoder) decode(sv reflect.Value, v reflect.Value) error {
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}

	if sv.Kind() == reflect.Ptr {
		sv = sv.Elem()
	}

	if sv.Kind() == reflect.Interface {
		if sv.IsNil() {
			return nil
		}
		sv = sv.Elem()
	}

	if !v.CanSet() {
		return errors.New("cannot set field " + v.String())
	}

	var e reflect.Type
	if v.Kind() == reflect.Ptr {
		e = v.Type().Elem()
	} else {
		e = v.Type()
	}

	switch e.Kind() {
	case reflect.Map:
		if sv.Kind() != reflect.Map {
			return errors.New("src not map")
		}
		t := v.Type()
		if t.Key().Kind() != reflect.String {
			return errors.New("dest<map> does not use string key")
		}
		if v.IsNil() {
			v.Set(reflect.MakeMap(t))
		}

		return d.mapCopy(sv, v)

	case reflect.Struct:
		// Decode map[string] interface{} into struct
		if sv.Kind() != reflect.Map {
			return errors.New("src not map")
		}
		if v.Kind() == reflect.Ptr && v.IsNil() {
			v.Set(reflect.New(e))
		}

		if v.Kind() == reflect.Ptr {
			return d.mapToStruct(sv, v.Elem())
		} else {
			return d.mapToStruct(sv, v)
		}

	case reflect.Slice, reflect.Array:
		if sv.Kind() != reflect.Slice {
			return errors.New("src not slice")
		}

		if v.Kind() == reflect.Slice && v.IsNil() {
			v.Set(reflect.MakeSlice(v.Type(), sv.Cap(), sv.Cap()))
		}

		return d.sliceCopy(sv, v)
	}

	if v.Kind() == reflect.Ptr {
		nv := reflect.New(e)
		err := d.decodeScalar(sv, nv)
		if err == nil {
			v.Set(nv)
		}
		return err
	}

	return d.decodeScalar(sv, v)
}

func (d *decoder) mapCopy(src reflect.Value, dst reflect.Value) error {
	sv := src.MapKeys()
	dt := dst.Type().Elem()

	for i := range sv {
		ot := reflect.New(dt)

		if dt.Kind() == reflect.Ptr {
			ot = reflect.New(ot.Type().Elem())
		}

		d.pushPath(sv[i].String())
		err := d.decode(src.MapIndex(sv[i]), ot)
		if err != nil {
			err = errors.New(d.pathString() + err.Error())
			d.path = nil
			return err
		}
		d.popPath()

		dst.SetMapIndex(sv[i], ot.Elem())
	}
	return nil
}

func (d *decoder) mapToStruct(src reflect.Value, dst reflect.Value) error {
	fieldIdx := make(map[string] int)
	for i := 0; i < dst.Type().NumField(); i++ {
		sf := dst.Type().Field(i)
		tag := sf.Tag.Get("json")
		if tag == "" {
			// do by name (only if exported field)
			if sf.Name[0] >= 'A' && sf.Name[0] <= 'Z' {
				fieldIdx[sf.Name] = i
			}
		} else if tag == "-" {
			// skip this field
		} else {
			fieldIdx[tag] = i
		}
	}

	sv := src.MapKeys()
	for i, k := range sv {
		if it, ok := fieldIdx[k.String()]; ok {
			d.pushPath(sv[i].String())
			err := d.decode(src.MapIndex(sv[i]), dst.Field(it))
			if err != nil {
				err = errors.New(d.pathString() + err.Error())
				d.path = nil
				return err
			}
			d.popPath()
		}
	}
	return nil
}

func (d *decoder) sliceCopy(src reflect.Value, dst reflect.Value) error {
	for i := 0; i < src.Len(); i++ {
		if i < dst.Len() {
			d.pushPath(fmt.Sprintf("[%d]", i))
			err := d.decode(src.Index(i), dst.Index(i))
			if err != nil {
				err = errors.New(d.pathString() + err.Error())
				d.path = nil
				return err
			}
			d.popPath()
		}
	}
	return nil
}


func (d *decoder) decodeScalar(src reflect.Value, dst reflect.Value) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	if dst.Kind() == reflect.Ptr {
		if dst.IsNil() {
			dst = reflect.New(dst.Type().Elem())
		}
		dst = dst.Elem()
	}

	switch dst.Kind() {
	case reflect.Bool:
		dst.SetBool(src.Bool())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		dst.SetInt(int64(src.Float()))

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		dst.SetUint(uint64(src.Float()))

	case reflect.Float32, reflect.Float64:
		dst.SetFloat(src.Float())

	case reflect.Interface:
		dst.Set(src)

	case reflect.String:
		if src.Kind() != reflect.String {
			return errors.New("not string kind: " + src.Kind().String())
		}
		dst.SetString(src.String())
	}

	return nil
}
