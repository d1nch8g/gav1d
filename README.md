# gav1d

AV1 video decoder for Go, powered by [dav1d](https://code.videolan.org/videolan/dav1d) via CGO.

## Features

- Decode AV1 video streams (IVF container)
- Multi-threaded decoding (`dav1d` SIMD optimizations: AVX2, SSE4)
- Returns raw YCbCr planes for each frame
- Single binary — no external dependencies at runtime

## Installation

```bash
go get github.com/d1nch8g/gav1d
```

## Requirements

- Go 1.21+
- GCC (for CGO)

## Usage

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/d1nch8g/gav1d"
)

func main() {
    data, _ := os.ReadFile("video.ivf")
    
    // Create decoder with 4 threads
    d, _ := gav1d.New(gav1d.Settings{Threads: 4})
    defer d.Close()
    
    // Decode all frames
    frames, _ := d.DecodeIVF(data)
    
    for i, f := range frames {
        fmt.Printf("Frame %d: %dx%d\n", i, f.Width(), f.Height())
        
        // Get YCbCr planes
        y, u, v, yStride, uvStride := f.YCbCr()
        _ = y // use planes for analysis / encoding
        
        f.Free()
    }
}
```

## How to get AV1 IVF files

Convert any video to AV1 IVF:

```bash
ffmpeg -i input.mp4 -t 10 -c:v copy -f ivf output.ivf
```

Or extract from WebM:

```bash
ffmpeg -i input.webm -c:v copy -f ivf output.ivf
```

## How it works

`gav1d` is a thin Go wrapper around `libdav1d`, the fastest AV1 decoder. The C library is precompiled and shipped with the Go package, so users don't need to install `dav1d` or `ffmpeg` — just `gcc`.

## Supported platforms

| Platform | Status |
|----------|--------|
| Linux x86_64 | ✅ Supported (AVX2/SSE4 SIMD) |
| Linux ARM64 | ⚠️ Needs rebuild without SIMD |
| macOS x86_64 | ⚠️ Untested |
| macOS ARM64 | ⚠️ Needs rebuild |
| Windows | ❌ Not yet |

## License

BSD 3-Clause — same as `dav1d`.

## Credits

- [dav1d](https://code.videolan.org/videolan/dav1d) — the fastest AV1 decoder
- Claude — helped with CGO bindings
