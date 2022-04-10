package cubone

import (
	"encoding/json"
	"io"
)

const (
	JsonCodec     string = "json"
	AvroCodec            = "avro"
	ProtobufCodec        = "proto"
)

type MessageCodec interface {
	Encode(v interface{}) ([]byte, error)
	EncodeToWriter(v interface{}, w io.Writer) error
	Decode(data []byte, v interface{}) error
	DecodeFromReader(v interface{}, r io.Reader) error
}

func NewDefaultCodec() MessageCodec {
	return &jsonCodec{}
}

type jsonCodec struct{}

func (c jsonCodec) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (c jsonCodec) EncodeToWriter(v interface{}, w io.Writer) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(v)
}

func (c jsonCodec) Decode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (c jsonCodec) DecodeFromReader(v interface{}, r io.Reader) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(v)
}
