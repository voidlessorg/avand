package main

import "encoding/base64"

func encB64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
func decB64(s string) []byte {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return make([]byte, 16)
	}
	return b
}
