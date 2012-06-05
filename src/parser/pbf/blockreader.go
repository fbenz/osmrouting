// Storage Layer - Parse OSMPBF blobs and decompress them.
// A .osm.pbf file consists of a series of blobs which are encoded as
// - header length (single int32 in big endian byte order)
// - BlobHeader (type + compressed data size)
// - BlobData (raw or zlib compressed block)
// We provide a single function (ReadBlock) to read a complete block from a
// given Reader.

// This file contains some code from go-osmpbf-filter, namely the readb function.
// (This means that if we ever release it, this will have to be replaced, or we
// will be subject to the GPLv3)

package pbf

import (
	"bytes"
	"code.google.com/p/goprotobuf/proto"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"../OSMPBF"
)

type BlockType int

const (
	OSMHeader BlockType = 0
	OSMData   BlockType = 1
)

type Block struct {
	Kind BlockType
	Data []byte
}

// Read a BlobHeader, including the size field
func readBlobHeader(stream io.Reader) (*OSMPBF.BlobHeader, error) {
	// Read the size of the blob header
	var headerSize int32
	if err := binary.Read(stream, binary.BigEndian, &headerSize); err != nil {
		return nil, err
	}

	// The OSMPBF specification prescribes that "The length of the BlobHeader
	// *should* be less than 32 KiB and *must* be less than 64 KiB."
	if headerSize < 0 || headerSize > (64*1024) {
		return nil, errors.New("Invalid blob header size")
	}

	buffer, err := readb(stream, headerSize)
	if err != nil {
		return nil, err
	}

	blobHeader := &OSMPBF.BlobHeader{}
	if err := proto.Unmarshal(buffer, blobHeader); err != nil {
		return nil, err
	}
	return blobHeader, nil
}

// Parse the blob type
func blobType(header *OSMPBF.BlobHeader) (BlockType, error) {
	t := proto.GetString(header.Type)
	switch t {
	case "OSMHeader":
		return OSMHeader, nil
	case "OSMData":
		return OSMData, nil
	}
	return 0, errors.New("Invalid BlobType: " + t)
}

// Read a pbf Blob, without decompressing
func readBlobData(stream io.Reader, header *OSMPBF.BlobHeader) (*OSMPBF.Blob, error) {
	// "The uncompressed length of a Blob *should* be less than 16 MiB
	// and *must* be less than 32 MiB. "
	blobSize := proto.GetInt32(header.Datasize)
	if blobSize < 0 || blobSize > (32*1024*1024) {
		return nil, errors.New("Invalid blob size")
	}

	buffer, err := readb(stream, blobSize)
	if err != nil {
		return nil, err
	}

	blob := &OSMPBF.Blob{}
	if err := proto.Unmarshal(buffer, blob); err != nil {
		return nil, err
	}
	return blob, nil
}

// Decompress a given blob
func decodeBlob(blob *OSMPBF.Blob) ([]byte, error) {
	// If the block is stored in uncompressed form
	if blob.Raw != nil {
		return blob.Raw, nil
	}

	// Otherwise we need to the uncompressed size
	if blob.RawSize == nil {
		return nil, errors.New("Compressed block without raw_size")
	}
	rawSize := *blob.RawSize

	if blob.ZlibData != nil {
		zlibBuffer := bytes.NewBuffer(blob.ZlibData)
		zlibReader, err := zlib.NewReader(zlibBuffer)
		if err != nil {
			return nil, err
		}
		contents, err := readb(zlibReader, rawSize)
		if err != nil {
			return nil, err
		}
		zlibReader.Close()
		return contents, nil
	}

	// The other formats (bzip2 and lzma) are not implemented in any encoder.
	return nil, errors.New("Unknown compression format")
}

// Read a Block from the beginning of the given stream.
func ReadBlock(stream io.Reader) (*Block, error) {
	blobHeader, err := readBlobHeader(stream)
	if err != nil {
		return nil, err
	}

	blobType, err := blobType(blobHeader)
	if err != nil {
		return nil, err
	}

	blobData, err := readBlobData(stream, blobHeader)
	if err != nil {
		return nil, err
	}

	rawData, err := decodeBlob(blobData)
	if err != nil {
		return nil, err
	}

	return &Block{blobType, rawData}, nil
}
