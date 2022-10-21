// Copyright 2022 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package prog

import (
	"bytes"
	"fmt"
	"math/rand"
)

type Heatmap interface {
	ChooseLocation(r *rand.Rand) int
}

type GenericHeatmap struct {
	segments  []segment // "Interesting" parts of the data.
	length    int       // Sum of all segment lengths.
	rawLength int       // Length of original data.
}

const granularity = 64 // Chunk size in bytes for processing the data.

func MakeGenericHeatmap(data []byte) Heatmap {
	var hm GenericHeatmap
	hm.rawLength = len(data)
	hm.length, hm.segments = calculateLengthAndSegments(data, granularity)
	return &hm
}

func (hm *GenericHeatmap) ChooseLocation(r *rand.Rand) int {
	if hm.length == 0 {
		// We have no segments, i.e. the data is all constant. Fall back to uniform selection.
		return r.Intn(hm.rawLength)
	}
	// Uniformly choose an index within one of the segments.
	heatmapIdx := r.Intn(hm.length)
	rawIdx := translateIdx(heatmapIdx, hm.segments)
	return rawIdx
}

type segment struct {
	offset int
	length int
}

func calculateLengthAndSegments(data []byte, granularity int) (int, []segment) {
	reader := bytes.NewReader(data)
	// Offset and length of current segment, total length of all segments.
	offset, currentLength, totalLength := 0, 0, 0
	segments := []segment{}
	buffer := make([]byte, granularity)

	for {
		n, err := reader.Read(buffer)
		if err != nil {
			break
		}

		// Check if buffer contains only a single value.
		byt0, isConstant := buffer[0], true
		for _, byt := range buffer[:n] {
			if byt != byt0 {
				isConstant = false
				break
			}
		}

		if !isConstant {
			// Non-constant - extend the current segment.
			currentLength += n
		} else {
			if currentLength != 0 {
				// Save current segment.
				segments = append(segments, segment{offset: offset, length: currentLength})
				offset, totalLength, currentLength = offset+currentLength, totalLength+currentLength, 0
			}
			// Skip past the constant bytes.
			offset += n
		}
	}

	// Save final segment.
	if currentLength != 0 {
		segments = append(segments, segment{offset: offset, length: currentLength})
		totalLength += currentLength
	}

	return totalLength, segments
}

func translateIdx(idx int, segs []segment) int {
	if idx < 0 {
		panic(fmt.Sprintf("translateIdx: negative index %v", idx))
	}
	savedIdx := idx
	for _, seg := range segs {
		if idx < seg.length {
			return seg.offset + idx
		}
		idx -= seg.length
	}
	panic(fmt.Sprintf("translateIdx: index out of range %v", savedIdx))
}
