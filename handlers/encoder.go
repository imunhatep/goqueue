package handlers

import (
	"bytes"
	"encoding/gob"
	"github.com/rs/zerolog/log"
)

type GobEncoder struct{}

func (g *GobEncoder) Encode(v interface{}) ([]byte, error) {
	log.Trace().Type("data", v).Msgf("[GobEncoder.Encode] encoding data: %T", v)
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
	log.Trace().Type("data", x).Msgf("[GobEncoder.Decode] decoding data: %T", x)

	return x, err
}
