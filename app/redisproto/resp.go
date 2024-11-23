package redisproto

import "strconv"

type Type byte

const (
	Integer    = ':'
	String     = '+'
	Error      = '-'
	Array      = '*'
	BulkString = '$'
)

// key components of a RESP message
type RESP struct {
	Type  Type    // Type of the RESP message
	Raw   []byte  // Raw encoded RESP message
	Data  []byte  // Decoded data (payload)
	Count int 	  // For arrays: number of elements
}

func (r RESP) ForEach(iter func(resp RESP) bool) {
	data := r.Data
	for i := 0; i < r.Count; i++ {
		n, resp := ReadNextRESP(data)
		if !iter(resp) {
			return
		}
		data = data[n:]
	}
}

func (r RESP) Bytes() []byte {
	return r.Data
}

func (r RESP) String() string {
	return string(r.Data)
}

func (r RESP) Int() int64 {
	x, _ := strconv.ParseInt(r.String(), 10, 64)
	return x
}

func (r RESP) Float() float64 {
	x, _ := strconv.ParseFloat(r.String(), 10)
	return x
}

// Map returns a key/value map of an Array.
// The receiver RESP must be an Array with an equal number of values, where
// the value of the key is followed by the key.
// Example: key1,value1,key2,value2,key3,value3
func (r RESP) Map() map[string]RESP {
	// ensuring this function only works for arrays
	if r.Type != Array {
		return nil
	}
	var n int // tracks the current index of the array
	var key string // temprarily stores the key as a string
	m := make(map[string]RESP) // initializing an empty map
	// iterate over each element of the array having a callback function
	r.ForEach(func(resp RESP) bool {
		// n = even ? store it as a key : store it as a value corresponding to the key
		if n&1 == 0 {
			key = resp.String()
		} else {
			m[key] = resp
		}
		n++
		return true
	})

	// returns the populated map
	return m
}


func (r RESP) MapGet(key string) RESP {
	if r.Type != Array {
		return RESP{}
	}
	var val RESP
	var n int
	var ok bool

	r.ForEach(func(resp RESP) bool {
		if n&1 == 0 {
			ok = resp.String() == key
		} else if ok {
			val = resp
			return false
		}
		n++
		return true
	})
	return val
}

func main() {}
