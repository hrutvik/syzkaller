// Copyright 2022 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package prog

import (
	"bytes"
	"fmt"
	"math/rand"
)

// Our heatmaps are a flexible mechanism to assign a probability distribution to
// some collection of bytes. Usage:
//  1. Choose a heatmap and initialize it: `hm := MakeXYZHeatmap(data)`.
//     Different heatmaps implement different probability distributions
//     (for now there is only one).
//  2. Select random indices according to the probability distribution:
//     `idx := hm.ChooseLocation(r)`.
type Heatmap interface {
	ChooseLocation(r *rand.Rand) int
}

// Generic heatmaps model a probability distribution based on sparse data,
// prioritising selection of regions which are not a single repeated byte. It
// views data as a series of chunks of length `granularity`, ignoring chunks
// which are a single repeated byte. Indices are chosen uniformly amongst the
// remaining "interesting" segments.
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

type GenericHeatmap struct {
	segments  []segment // "Interesting" parts of the data.
	length    int       // Sum of all segment lengths.
	rawLength int       // Length of original data.
}

type segment struct {
	offset int
	length int
}

const granularity = 64 // Chunk size in bytes for processing the data.

// Determine the "interesting" segments of data, also returning their combined length.
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

// Convert from an index into "interesting" segments to an index into raw data.
// I.e. view `idx` as an index into the concatenated segments, and translate
// this to an index into the original underlying data. E.g.:
//
//	segs = []segment{{offset: 10, length: 20}, {offset: 50, length: 10}}
//	translateIdx(25, segs) = 5
//
// I.e. we index element 5 of the second segment, so element 55 of the raw data.
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
