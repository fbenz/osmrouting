package osm

// Storage Layer - Parse osmpbf blobs and decompress them.
// A .osm.pbf file consists of a series of blobs which are encoded as
// - header length (single int32 in big endian byte order)
// - BlobHeader (type + compressed data size)
// - BlobData (raw or zlib compressed block)
// We provide a single function (ReadBlock) to read a complete block from a
// given Reader.

import (
	"bytes"
	"code.google.com/p/goprotobuf/proto"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"osm/pbf"
)

const (
	MaxHeaderSize = 64 << 10
	MaxBlobSize   = 32 << 20
)

type BlockType int

const (
	OSMHeader BlockType = iota
	OSMData
)

type Block struct {
	Kind BlockType
	Data []byte
}

// Read exactly the number of requested bytes.
// I don't know if this is truly necessary, but it is included in go-osmpbf-filter...
func readb(file io.Reader, size int32) ([]byte, error) {
	buffer := make([]byte, size)
	var idx int32 = 0
	for {
		cnt, err := file.Read(buffer[idx:])
		if err != nil {
			return nil, err
		}
		idx += int32(cnt)
		if idx == size {
			break
		}
	}
	return buffer, nil
}

// Read a BlobHeader, including the size field
func readBlobHeader(stream io.Reader) (*pbf.BlobHeader, error) {
	// Read the size of the blob header
	var headerSize int32
	if err := binary.Read(stream, binary.BigEndian, &headerSize); err != nil {
		return nil, err
	}

	if headerSize < 0 || headerSize > MaxHeaderSize {
		return nil, errors.New("Invalid blob header size")
	}

	buffer, err := readb(stream, headerSize)
	if err != nil {
		return nil, err
	}

	blobHeader := &pbf.BlobHeader{}
	if err := proto.Unmarshal(buffer, blobHeader); err != nil {
		return nil, err
	}
	return blobHeader, nil
}

// Parse the blob type
func blobType(header *pbf.BlobHeader) (BlockType, error) {
	t := header.GetType()
	switch t {
	case "OSMHeader":
		return OSMHeader, nil
	case "OSMData":
		return OSMData, nil
	}
	return 0, errors.New("Invalid BlobType: " + t)
}

// Read a pbf Blob, without decompressing it
func readBlobData(stream io.Reader, header *pbf.BlobHeader) (*pbf.Blob, error) {
	blobSize := header.GetDatasize()
	if blobSize < 0 || blobSize > MaxBlobSize {
		return nil, errors.New("Invalid blob size")
	}

	buffer, err := readb(stream, blobSize)
	if err != nil {
		return nil, err
	}

	blob := &pbf.Blob{}
	if err := proto.Unmarshal(buffer, blob); err != nil {
		return nil, err
	}
	return blob, nil
}

// Decompress a given blob
func decodeBlob(blob *pbf.Blob) ([]byte, error) {
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
