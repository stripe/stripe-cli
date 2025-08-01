package useragent

import "bytes"

func trimNulls(input []byte) []byte {
	return bytes.Trim(input, "\x00")
}
