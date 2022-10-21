// Copyright 2022 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package prog

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
)

func Compress(rawData []byte) ([]byte, error) {
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)

	_, err := gzipWriter.Write(rawData)
	if err != nil {
		return nil, fmt.Errorf("could not compress with gzip: %v", err)
	}

	err = gzipWriter.Close()
	return buffer.Bytes(), err
}

func Decompress(compressedData []byte) ([]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, fmt.Errorf("could not initialise gzip: %v", err)
	}

	data, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, fmt.Errorf("could not read data with gzip: %v", err)
	}

	err = gzipReader.Close()
	return data, err
}

func DecodeB64(b64Data []byte) ([]byte, error) {
	decoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(b64Data))
	rawData, err := io.ReadAll(decoder)
	if err != nil {
		return nil, fmt.Errorf("could not decode Base64: %v", err)
	}
	return rawData, nil
}

func EncodeB64(rawData []byte) ([]byte, error) {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	_, err := encoder.Write(rawData)
	if err != nil {
		return nil, fmt.Errorf("could not encode Base64: %v", err)
	}
	encoder.Close()
	return buf.Bytes(), nil
}
