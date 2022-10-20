// Copyright 2022 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package prog

import (
	"bytes"
	"fmt"
)

type Heatmap interface {
	Populate(data []byte)
	ChooseLocation(r *randGen) int
}

type GenericHeatmap struct {
	segments []segment // "Interesting" parts of the data.
	length   int       // Sum of all segment lengths.
}

func (hm *GenericHeatmap) Populate(data []byte) {
	const granularity = 64 // Chunk size in bytes for processing the data.
	hm.length, hm.segments = calculateLengthAndSegments(data, granularity)
}

func (hm *GenericHeatmap) ChooseLocation(r *randGen) int {
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
	offset, length := 0, 0 // Offset and length of current segment.
	segments := []segment{}
	buffer := make([]byte, granularity)

	for n, err := reader.Read(buffer); n > 0 && err == nil; {
		// Check if buffer contains only a single value.
		byt0, isConstant := buffer[0], true
		for _, byt := range buffer[:n] {
			if byt != byt0 {
				isConstant = false
				break
			}
		}

		if !isConstant {
			length += n
		} else {
			if length != 0 {
				// Save current segment.
				segments = append(segments, segment{offset: offset, length: length})
				length = 0
			}
			offset += length + n
		}
	}

	// Save final segment.
	if length != 0 {
		segments = append(segments, segment{offset: offset, length: length})
		offset += length
	}

	return offset, segments
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
