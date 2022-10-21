// Copyright 2022 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package prog

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

type Region struct {
	start int
	end   int
}

func TestGenericHeatmap(t *testing.T) {
	const tries = 10
	iters := iterCount() / tries

	r := rand.New(randSource(t))
	for i := 0; i < iters; i++ {
		data, regions := createData(r)
		hm := MakeGenericHeatmap(data).(*GenericHeatmap)

		for j := 0; j < tries; j++ {
			index := hm.ChooseLocation(r)
			if !checkIndex(index, len(data), regions) {
				t.Fatalf("selected index %d does not fall in a region\n", index)
				hm.debugPrint(data, regions)
			}
		}
	}
}

// Create a byte slice which is mostly a single byte. Return the data and the regions we want the heatmap to select.
func createData(r *rand.Rand) ([]byte, []Region) {
	// Initialise slice over 128 KB and up to ~1 MB in length.
	len := r.Intn(1<<20) + 1<<17
	data := make([]byte, len)

	// Fill slice with constant byte.
	constByte := byte(r.Intn(256))
	for i := range data {
		data[i] = constByte
	}

	// Randomly select "interesting" regions. We don't care about overlapping or empty regions.
	numRegions := r.Intn(10)
	var regions []Region
	for i := 0; i < numRegions; i++ {
		start := r.Intn(len)
		length := r.Intn((len / 100) + 1)
		if start+length > len {
			continue
		}
		regions = append(regions, Region{start, start + length})
	}

	// Fill "interesting" regions with random data.
	for _, region := range regions {
		r.Read(data[region.start:region.end])
	}
	return data, regions
}

// Check an index is within some (aligned) region.
func checkIndex(index, maxIndex int, regions []Region) bool {
	if index < 0 || index >= maxIndex {
		return false
	}

	if len(regions) == 0 {
		// Index is chosen uniformly up to maxIndex.
		return true
	}
	for _, region := range regions {
		start, end := roundDown(region.start), roundDown(region.end)+granularity
		if start <= index && index < end {
			return true
		}
	}
	return false
}

func roundDown(i int) int {
	return (i / granularity) * granularity
}

func (hm *GenericHeatmap) debugPrint(data []byte, regions []Region) {
	// Print data.
	fmt.Printf("data: len = %d\n", len(data))
	for j := 0; j < len(data); j += granularity {
		end := j + granularity
		if end > len(data) {
			end = len(data)
		}
		fmt.Printf("%8d: %x\n", j*granularity, data[j:end])
	}
	fmt.Println()

	// Print selected regions in data.
	sort.Slice(regions, func(i, j int) bool {
		return regions[i].start < regions[j].start
	})
	for j, region := range regions {
		fmt.Printf("region  %4d: %8v - %8v\n", j, region.start, region.end)
	}
	fmt.Println()

	// Print heatmap.
	fmt.Printf("Heatmap (total segment length %d, total length %d)\n", hm.length, hm.rawLength)
	for j, seg := range hm.segments {
		fmt.Printf("segment %4d: %8v - %8v\n", j, seg.offset, seg.offset+seg.length)
	}
	fmt.Printf("\n\n\n")
}
