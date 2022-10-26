/*
	Decompression using zlib.
	Derived from https://zlib.net/zpipe.c, by Mark Adler (in the public domain).
*/

#include "zlib.h"
#include <assert.h>
#include <stdio.h>
#include <string.h>

// We have to keep this quite small to avoid hitting frame size of 16384 bytes.
#define CHUNK 4096

void cleanup(z_stream* stream, FILE* source)
{
	inflateEnd(stream);
	fclose(source);
}

/*
	Decompress from input 'array' of 'length' to 'dest' file.  Returns Z_OK on
	success, Z_MEM_ERROR for OOM, Z_DATA_ERROR for corrupt data,
	Z_VERSION_ERROR if zlib.h and library versions do not match, and Z_ERRNO
	for read/write errors.
*/
int decompress(unsigned char* input, size_t length, FILE* dest)
{
	int ret;
	unsigned int have;
	z_stream stream;
	unsigned char in[CHUNK];
	unsigned char out[CHUNK];

	// Allocate inflate state.
	stream.zalloc = Z_NULL;
	stream.zfree = Z_NULL;
	stream.opaque = Z_NULL;
	stream.avail_in = 0;
	stream.next_in = Z_NULL;
	ret = inflateInit(&stream);
	if (ret != Z_OK)
		return ret;

	// Create source stream.
	FILE* source = fmemopen(input, length, "r");

	// Decompress until deflate stream ends or EOF.
	do {
		stream.avail_in = fread(in, 1, CHUNK, source);
		if (ferror(source)) {
			cleanup(&stream, source);
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
				cleanup(&stream, source);
				return ret;
			}
			have = CHUNK - stream.avail_out;
			if (fwrite(out, 1, have, dest) != have || ferror(dest)) {
				cleanup(&stream, source);
				return Z_ERRNO;
			}
		} while (stream.avail_out == 0);

	} while (ret != Z_STREAM_END); // Wait for inflate() to report it is finished.

	cleanup(&stream, source);
	return ret == Z_STREAM_END ? Z_OK : Z_DATA_ERROR;
}
