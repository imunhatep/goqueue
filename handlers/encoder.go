package handlers

import (
	"bytes"
	"encoding/gob"
)

type GobEncoder struct{}

func (g *GobEncoder) Encode(v interface{}) ([]byte, error) {
	var x interface{} = v

	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	err := enc.Encode(&x)

	return buf.Bytes(), err
}

func (g *GobEncoder) Decode(data []byte) (interface{}, error) {
	var x interface{}

	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&x)

	return x, err
}
