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
	Type  Type   // Type of the RESP message
	Raw   []byte // Raw encoded RESP message
	Data  []byte // Decoded data (payload)
	Count int    // For arrays: number of elements
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
	var n int                  // tracks the current index of the array
	var key string             // temprarily stores the key as a string
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

// function for retrieving the value associated with a specific key
// from the RESP object received from the client containing an array of key-value pairs
// key, in the form of a string, goes in the input
// function only works with RESP arrays
func (r RESP) MapGet(key string) RESP {
	// checking whether r is of type Array
	// if not, returns an empty RESP
	if r.Type != Array {
		return RESP{}
	}
	var val RESP // holds the value associated with a particular key in the RESP object
	var n int    // tracks the current index in the RESP array
	var ok bool  // a boolean flag indicating whether the current key matches the given key

	// iterating over each element in the RESP array
	// with a callback function
	r.ForEach(func(resp RESP) bool {
		// when n is even, the current RESP element is treated as a key
		if n&1 == 0 {
			// key is converted into a string and ok turns into true
			// if the current key and the given key match, then ok == true
			// otherwise ok == false
			ok = resp.String() == key
		} else if ok {
			// n is odd and ok == true, then value corresponding to the given key is stored in val
			// and the iteration stops
			val = resp
			return false
		}
		n++
		return true
	})
	return val
}

func (r RESP) Exists() bool {
	return r.Type != 0
}


//function designed to parse and decode a RESP message from a byte slice 'b'
// function identifies the type of RESP data(like integer, bulk string, array, etc) and extracts
// its content for further processing.
// b = A byte slice containing a RESP message
// n = Number of bytes consumed from the input slice to parse the message
// resp = a RESP struct containing the parsed type, raw data, and extracted data
func ReadNextRESP(b []byte) (n int, resp RESP) {
	// if the input byte slice is empty, then 0 and empty RESP is returned
	if len(b) == 0 {
		return 0, RESP{}
	}

	// first byte determines the type like Integer, String, BulkString, or any other
	// any other type, other than that which is defined in Type, returns 0 and empty RESP
	resp.Type = Type(b[0])
	switch resp.Type {
	case Integer, String, BulkString, Array, Error:
	default:
		return 0, RESP{}
	}
	i := 1
	for ; ; i++ {
		// input i ends prematurely
		if i == len(b) {
			return 0, RESP{}
		}
		// if '\n' is located in the input byte slice
		if b[i] == '\n' {
			// if not '\r\n', return 0 and empty RESP
			if b[i-1] != '\r' {
				return 0, RESP{}
			}
			// otherwise, after locating '\r\n', it breaks the loop
			i++
			break
		}
	}

	// raw RESP message upto '\r\n'
	resp.Raw = b[0:i]
	// data RESP message: actual content after the type byte and before '\r\n'
	resp.Data = b[1 : i-2]

	// if the type of the actual content received is INTEGER
	// example, :123\r\n represents the integer 123
	// prefixed by ':'
	if resp.Type == Integer {
		// ensuring that resp data is non-empty
		if len(resp.Data) == 0 {
			return 0, RESP{}
		}
		// checking for negative numbers like, :-123\r\n
		// data after parsing is '-123'
		// also, ensuring that resp.Data is non-empty i.e., having only '-' e.g., ':-\r\n'
		var j int
		if resp.Data[0] == '-' {
			if len(resp.Data) == 1 {
				return 0, RESP{}
			}
			// if there is a '-' sign, j starts from 1 i.e., from the actual data
			j++
		}

		// loop starting from 0 in case of positive numbers
		// and from 1 in case of negative numbers
		// valid: 123, -456
		// invalid: 12a3, 45.67
		for ; j < len(resp.Data); j++ {
			if resp.Data[j] < '0' || resp.Data[j] > '9' {
				return 0, RESP{}
			}
		}

		// finally, the total number of bytes consumed from the input (b)
		// from ':' to the CRLF terminator '\r\n' is returned,
		// along with the RESP object, containing the parsed integer in resp.Data
		return len(resp.Raw), resp
	}

	// if the type is String or Error, the function simply returns the parsed RESP
	// without further processing
	if resp.Type == String || resp.Type == Error {
		return len(resp.Raw), resp
	}

	// for the case of receiving BULK STRING
	// Redis encodes a bulk string in the following way:
	// $<length>\r\n<data>\r\n
	// example, for $6\r\nfoobar\r\n, length = 6 and data = foobar
	// resp.Data = 6
	// strconv.Atoi(resp.Data) parses the string, "6", into integer, 6
	// integer parsed from the string is stored in the resp.Count
	var err error
	resp.Count, err = strconv.Atoi(string(resp.Data))
	if resp.Type == BulkString {
		// handles the error in case of conversion
		if err != nil {
			return 0, RESP{}
		}
		// Null BulkString
		// invalid data in resp.Data, i.e., it could not be parsed into integer
		// $-1\r\n = Null BulkString
		if resp.Count < 0 {
			resp.Data = nil // absence of content
			resp.Count = 0 // setting resp.Count to 0
			return len(resp.Raw), resp // returns the parsed RESP
		}
		// assuring enough data in the bulk string
		// length of the prefix: i
		// length of data: resp.Count i.e., the parsed integer
		// length of ending: 2 i.e., \r\n
		// in case len(b) is less that the sum of the above three, then it indicates invalid input
		if len(b) < i+resp.Count+2 {
			return 0, RESP{} // retuned in case of an error
		}
		// ensures that the data ends with '\r\n' i.e., CRLF terminator
		if b[i+resp.Count] != '\r' || b[i+resp.Count+1] != '\n' {
			return 0, RESP{}
		}
		// puts the actual data of the bulk string into Data
		// for '$6\r\nfoobar\r\n', resp.Data = 'foobar'
		resp.Data = b[i : i+resp.Count]
		// for '$6\r\nfoobar\r\n', resp.Count = '6\r\nfoobar\r\n'
		resp.Raw = b[0 : i+resp.Count+2]
		resp.Count = 0

		// returns the length of the bytes processed from $ to the final \r\n
		// also, returns the resp object containing resp.Data and resp.Raw
		return len(resp.Raw), resp
	}

	// for the case of 'ARRAYS'
	// Redis encodes an array in the following way:
	// *<number-of-elements>\r\n<element-1>\r\n...<element-n>\r\n
	// Asterisk(*) as the first byte
	// empty array looks like: *0\r\n
	// encoding of an array having two bulk strings "hello" and "world" is as follows:
	// *2\r\n$5\r\nhello\r\n$5\r\nworld\r\n
	if err != nil {
		return 0, RESP{}
	}
	var tn int
	sData := b[i:]
	for j := 0; j < resp.Count; j++ {
		rn, rresp := ReadNextRESP(sData)
		// if the parsing of the array element fails i.e., 0, then an empty resp and 0 is returned
		if rresp.Type == 0 {
			return 0, RESP{} // signalling an incomplete or invalid array
		}

		// processed bytes are updated and added continuously
		tn += rn

		// sData is moved to the next array element
		// by the length of the bytes of the previous array element
		sData = sData[rn:]
	}

	// raw data of the entire array is stored in resp.Data
	// resp.Data contains the parsed data
	resp.Data = b[i : i+tn]
	// sliced the buffer (b) to include the full RESP array, starting from * to the end of last element
	// resp.Raw preserves the exact byte sequence of the RESP message for debugging or retransmission
	resp.Raw = b[0 : i+tn]
	return len(resp.Raw), resp
}

func main() {}
