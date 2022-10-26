// Copyright 2022 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.
//
// Decompression using zlib.
// Derived from https://zlib.net/zpipe.c, by Mark Adler (in the public domain).

#include <assert.h>
#include <stdio.h>
#include <string.h>
#include <zlib.h>

// We have to keep this quite small to avoid hitting frame size of 16384 bytes.
#define CHUNK 4096

static void cleanup(z_stream* stream, FILE* source, FILE* dest)
{
	inflateEnd(stream);
	fclose(source);
	fclose(dest);
}

// Decompress from 'input' array of 'length' to 'dest_fd' file descriptor.
// Returns Z_OK on success, Z_MEM_ERROR for OOM, Z_DATA_ERROR for corrupt data,
// Z_VERSION_ERROR if zlib.h and library versions do not match, and Z_ERRNO
// for read/write errors.
static int decompress(unsigned char* input, size_t length, int dest_fd)
{

	unsigned char in[CHUNK];
	unsigned char out[CHUNK];

	// Allocate inflate state.
	z_stream stream;
	stream.zalloc = Z_NULL;
	stream.zfree = Z_NULL;
	stream.opaque = Z_NULL;
	stream.avail_in = 0;
	stream.next_in = Z_NULL;
	int ret = inflateInit2(&stream, 16 + MAX_WBITS); // Decompress gzip.
	if (ret != Z_OK)
		return ret;

	// Create source stream.
	FILE* source = fmemopen(input, length, "rb");
	if (errno)
		return Z_ERRNO;

	// Create destination stream.
	FILE* dest = fdopen(dest_fd, "wb");
	if (errno)
		return Z_ERRNO;

	// Decompress until deflate stream ends or EOF.
	do {
		stream.avail_in = fread(in, 1, CHUNK, source);
		if (ferror(source)) {
			cleanup(&stream, source, dest);
			return Z_ERRNO;
		}
		if (stream.avail_in == 0)
			break;
		stream.next_in = in;

		// inflate() input until output buffer is full.
		do {
			stream.avail_out = CHUNK;
			stream.next_out = out;
			ret = inflate(&stream, Z_NO_FLUSH);
			assert(ret != Z_STREAM_ERROR); // state not clobbered
			switch (ret) {
			case Z_NEED_DICT:
				ret = Z_DATA_ERROR; // and fall through
			case Z_DATA_ERROR:
			case Z_MEM_ERROR:
				cleanup(&stream, source, dest);
				return ret;
			}
			unsigned int have = CHUNK - stream.avail_out;
			if (fwrite(out, 1, have, dest) != have || ferror(dest)) {
				cleanup(&stream, source, dest);
				return Z_ERRNO;
			}
		} while (stream.avail_out == 0);

	} while (ret != Z_STREAM_END); // Wait for inflate() to report it is finished.

	cleanup(&stream, source, dest);
	return ret == Z_STREAM_END ? Z_OK : Z_DATA_ERROR;
}
