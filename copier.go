// BSD licensed

package interfacetools

import (
	"errors"
	"reflect"
	"bytes"
	"fmt"
	"strconv"
)

// Copier is the interface implemented by objects
// that can decode input interfaces themselves.
// The Copier must copy the input data if it wishes
// to retain the data after returning. Unhandled
// input must return an error.
type Copier interface {
	CopyIn(interface{}) error
}

// Performs a deep-copy of source object to the passed in interface.
//
// If the object is a struct, then the copy is attempted based on the tag
// of the json tag of the field; the same rules apply per encoding/json.
// If a field is not available, then a copier skips it.
func CopyOut(src interface{}, out interface{}) (err error) {
	return CopyOutTagged(src, out, "json")
}

func CopyOutTagged(src interface{}, out interface{}, tag string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr {
		return errors.New("output interface must be a pointer")
	}

	dec := newDecoder(tag)
	iv := reflect.ValueOf(src)
	ov := reflect.ValueOf(out)
	return dec.decode(iv, ov)
}

type decoder struct {
	tag  string
	path []string
}

func newDecoder(tag string) *decoder {
	return &decoder{tag, make([]string, 0, 12)}
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

func (d *decoder) tryCopyIn(sv reflect.Value, v reflect.Value) bool {
	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}
	for {
		// Load value from interface, but only if the result will be
		// usefully addressable.
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() && e.Elem().Kind() == reflect.Ptr {
				v = e
				continue
			}
		}
		if v.Kind() != reflect.Ptr {
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if v.Type().NumMethod() > 0 {
			if u, ok := v.Interface().(Copier); ok {
				// Can now attempt CopyIn
				if u.CopyIn(sv.Interface()) == nil {
					return true
				}
			}
		}
		v = v.Elem()
	}

	return false
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
		if d.tryCopyIn(sv, v) {
			return nil
		}

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
		if d.tryCopyIn(sv, v) {
			return nil
		}

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
		if d.tryCopyIn(sv, v) {
			return nil
		}

		if sv.Kind() != reflect.Slice {
			return errors.New("src not slice")
		}

		if v.Kind() == reflect.Slice && v.IsNil() {
			v.Set(reflect.MakeSlice(v.Type(), sv.Len(), sv.Len()))
		}

		return d.sliceCopy(sv, v)
	}

	if v.Kind() == reflect.Ptr {
		nv := reflect.New(e)

		if d.tryCopyIn(sv, nv) {
			return nil
		}

		err := d.decodeScalar(sv, nv.Elem())
		if err == nil {
			v.Set(nv)
		}
		return err
	}

	if d.tryCopyIn(sv, v) {
		return nil
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

		if sf.Anonymous {
			// Dig into anonymous field and attempt decode at current
			// level there. If it is a pointer, allocate the element.
			dv := dst.Field(i)
			if dv.Kind() == reflect.Ptr && dv.IsNil() {
				dv.Set(reflect.New(dv.Type().Elem()))
				dv = dv.Elem()
			}
			err := d.mapToStruct(src, dv)
			if err != nil {
				return err
			}
		}

		tag := sf.Tag.Get(d.tag)
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

	switch dst.Kind() {
	case reflect.Bool:
		dst.SetBool(src.Bool())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch src.Kind() {
		case reflect.String:
			n, err := strconv.ParseInt(src.String(), 10, 64)
			if err != nil {
				return err
			}
			dst.SetInt(n)

		case reflect.Float32, reflect.Float64:
			dst.SetInt(int64(src.Float()))

		case reflect.Int, reflect.Int8, reflect.Int16,
		     reflect.Int32, reflect.Int64:
			dst.SetInt(src.Int())

		case reflect.Uint, reflect.Uint8, reflect.Uint16,
		     reflect.Uint32, reflect.Uint64:
			dst.SetInt(int64(src.Uint()))

		default:
			return errors.New(fmt.Sprintf("Cannot convert source kind %v to destination kind %v", src.Kind(), dst.Kind()))
		}


	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch src.Kind() {
		case reflect.String:
			n, err := strconv.ParseUint(src.String(), 10, 64)
			if err != nil {
				return err
			}
			dst.SetUint(n)

		case reflect.Float32, reflect.Float64:
			dst.SetUint(uint64(src.Float()))

		case reflect.Int, reflect.Int8, reflect.Int16,
		     reflect.Int32, reflect.Int64:
			dst.SetUint(uint64(src.Int()))

		case reflect.Uint, reflect.Uint8, reflect.Uint16,
		     reflect.Uint32, reflect.Uint64:
			dst.SetUint(src.Uint())

		default:
			return errors.New(fmt.Sprintf("Cannot convert source kind %v to dest kind %v", src.Kind(), dst.Kind()))
		}

	case reflect.Float32, reflect.Float64:
		dst.SetFloat(src.Float())
		switch src.Kind() {
		case reflect.String:
			n, err := strconv.ParseFloat(src.String(), 64)
			if err != nil {
				return err
			}
			dst.SetFloat(n)

		case reflect.Float32, reflect.Float64:
			dst.SetFloat(src.Float())

		case reflect.Int, reflect.Int8, reflect.Int16,
		     reflect.Int32, reflect.Int64:
			dst.SetFloat(float64(src.Int()))

		case reflect.Uint, reflect.Uint8, reflect.Uint16,
		     reflect.Uint32, reflect.Uint64:
			dst.SetFloat(float64(src.Uint()))

		default:
			return errors.New(fmt.Sprintf("Cannot convert source kind %v to dest kind %v", src.Kind(), dst.Kind()))
		}

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
