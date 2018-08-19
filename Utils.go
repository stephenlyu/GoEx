package goex

import (
	"strconv"
	"reflect"
)


func ToFloat64(i interface{}) float64 {
	value := reflect.ValueOf(i)
	switch i.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return float64(value.Int())
	case float32, float64:
		return value.Float()
	case string:
		vStr := i.(string)
		vF, _ := strconv.ParseFloat(vStr, 64)
		return vF
	default:
		panic("Unreachable code")
	}
	return 0
}

func ToUint64(i interface{}) uint64 {
	value := reflect.ValueOf(i)
	switch i.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return uint64(value.Int())
	case float32, float64:
		return uint64(value.Float())
	case string:
		uV, _ := strconv.ParseUint(i.(string), 10, 64)
		return uV
	default:
		panic("Unreachable code")
	}
	return 0
}

func ToInt(i interface{}) int {
	value := reflect.ValueOf(i)
	switch i.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return int(value.Int())
	case float32, float64:
		return int(value.Float())
	case string:
		vStr := i.(string)
		vInt, _ := strconv.Atoi(vStr)
		return vInt
	default:
		panic("Unreachable code")
	}
	return 0
}
