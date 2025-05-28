package handlers

import (
	"encoding/gob"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGobEncoder_EncodeDecode(t *testing.T) {
	encoder := &GobEncoder{}
	gob.Register(map[string]interface{}{})

	// Test data
	originalData := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": []string{"a", "b", "c"},
	}

	// Test Encode
	encodedData, err := encoder.Encode(originalData)
	assert.NoError(t, err, "Encode should not return an error")
	assert.NotNil(t, encodedData, "Encoded data should not be nil")

	// Test Decode
	decodedData, err := encoder.Decode(encodedData)
	assert.NoError(t, err, "Decode should not return an error")
	assert.Equal(t, originalData, decodedData, "Decoded data should match the original data")
}

func TestGobEncoder_EncodeError(t *testing.T) {
	encoder := &GobEncoder{}

	// Test encoding an unsupported type (e.g., a channel)
	unsupportedData := make(chan int)

	_, err := encoder.Encode(unsupportedData)
	assert.Error(t, err, "Encode should return an error for unsupported types")
}

func TestGobEncoder_DecodeError(t *testing.T) {
	encoder := &GobEncoder{}

	// Test decoding invalid data
	invalidData := []byte("invalid gob data")

	_, err := encoder.Decode(invalidData)
	assert.Error(t, err, "Decode should return an error for invalid data")
}
