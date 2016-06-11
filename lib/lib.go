package tracerlib

import (
	"errors"
	"strconv"
)

var ErrTypeConversion = errors.New("Unable to convert to type string")
var ErrNoKey = errors.New("Did not find the specified key in map")
var ErrNoOutputFound = errors.New("Did not find the specified trx: outptut")

// GetFromMapInterface ( "op", dm )
func GetFromMapInterface(key string, data map[string]interface{}) (rv string, err error) {
	v1, ok := data[key]
	if ok {
		v2, ok := v1.(string)
		if ok {
			rv = v2
			return
		}
		err = ErrTypeConversion
		return
	}
	err = ErrNoKey
	return
}

// GetFromMapInterface ( "op", dm )
func GetInt64FromMapInterface(key string, data map[string]interface{}) (rv int64, err error) {
	v1, ok := data[key]
	if ok {
		//fmt.Printf("AT %s - have value for key %s, Type=%T\n", godebug.LF(), key, v1)
		v2, ok := v1.(float64)
		if ok {
			//fmt.Printf("AT %s\n", godebug.LF())
			rv = int64(v2)
			return
		} else {
			// try a string
			v3, ok := v1.(string)
			if ok {
				t, xerr := strconv.ParseInt(v3, 10, 64)
				if xerr != nil {
					err = xerr
					return
				}
				rv = t
			}
		}
		err = ErrTypeConversion
		return
	}
	err = ErrNoKey
	return
}

// mm_Pw := tracerlib.QueryGet(u.Query, "Pw")
