package pbf

import (
	"io"
)

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
