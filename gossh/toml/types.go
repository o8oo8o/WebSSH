package toml

import (
	"encoding"
	"reflect"
	"time"
)

var timeType = reflect.TypeOf((*time.Time)(nil)).Elem()
var textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
var textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
var mapStringInterfaceType = reflect.TypeOf(map[string]any(nil))
var sliceInterfaceType = reflect.TypeOf([]any(nil))
var stringType = reflect.TypeOf("")
