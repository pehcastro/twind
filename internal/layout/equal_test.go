package layout

import (
	"reflect"
	"testing"
)

func nonZero(v reflect.Value) {
	switch v.Kind() {
	case reflect.Struct:
		nonZero(v.Field(0))
	case reflect.Slice:
		v.Set(reflect.MakeSlice(v.Type(), 1, 1))
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int64:
		v.SetInt(1)
	case reflect.Uint8:
		v.SetUint(1)
	default:
		panic("equal_test: no non-zero value for " + v.Kind().String())
	}
}

func TestStyleEqualSeesEveryField(t *testing.T) {
	var zero Style
	if !zero.Equal(&zero) {
		t.Fatal("a zero style is not equal to itself")
	}
	for i := range reflect.TypeFor[Style]().NumField() {
		var a Style
		nonZero(reflect.ValueOf(&a).Elem().Field(i))
		if a.Equal(&zero) || zero.Equal(&a) {
			t.Errorf("styles differing in %s are equal", reflect.TypeFor[Style]().Field(i).Name)
		}
		if b := a; !a.Equal(&b) {
			t.Errorf("a style set in %s is not equal to its copy", reflect.TypeFor[Style]().Field(i).Name)
		}
	}
	a, b := Style{Columns: []Track{{Min: Breadth{Kind: SizeFr, Value: 1}}}}, Style{Columns: []Track{{Min: Breadth{Kind: SizeFr, Value: 2}}}}
	if a.Equal(&b) {
		t.Error("styles whose first column differs are equal")
	}
	if empty := (Style{Rows: []Track{}}); !empty.Equal(&zero) {
		t.Error("an empty row list differs from none")
	}
}
