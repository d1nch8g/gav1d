package gav1d

/*
#cgo CFLAGS: -I${SRCDIR}/include/dav1d -I${SRCDIR}/build
#cgo LDFLAGS: -L${SRCDIR}/build/src -ldav1d
#include <dav1d/dav1d.h>
#include <stdlib.h>
#include <errno.h>

static void free_callback(const uint8_t *buf, void *cookie) {
    free((void *)buf);
}

static int wrap_data(Dav1dData *data, uint8_t *buf, size_t sz) {
    return dav1d_data_wrap(data, buf, sz, free_callback, NULL);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type Decoder struct {
	ctx *C.Dav1dContext
}

type Settings struct {
	Threads int
}

func New(settings Settings) (*Decoder, error) {
	var s C.Dav1dSettings
	C.dav1d_default_settings(&s)
	s.n_threads = C.int(settings.Threads)

	var ctx *C.Dav1dContext
	ret := C.dav1d_open(&ctx, &s)
	if ret != 0 {
		return nil, fmt.Errorf("dav1d_open: %d", ret)
	}
	return &Decoder{ctx: ctx}, nil
}

func (d *Decoder) SendData(data []byte) error {
	var pkt C.Dav1dData
	// CBytes делает malloc-копию — free_callback её освободит
	cbuf := (*C.uint8_t)(C.CBytes(data))
	ret := C.wrap_data(&pkt, cbuf, C.size_t(len(data)))
	if ret != 0 {
		C.free(unsafe.Pointer(cbuf))
		return fmt.Errorf("dav1d_data_wrap: %d", ret)
	}
	ret2 := C.dav1d_send_data(d.ctx, &pkt)
	if ret2 < 0 && ret2 != -C.EAGAIN {
		C.dav1d_data_unref(&pkt)
		return fmt.Errorf("dav1d_send_data: %d", ret2)
	}
	return nil
}

func (d *Decoder) GetPicture() (*Frame, error) {
	var pic C.Dav1dPicture
	ret := C.dav1d_get_picture(d.ctx, &pic)
	if ret == -C.EAGAIN {
		return nil, nil
	}
	if ret < 0 {
		return nil, fmt.Errorf("dav1d_get_picture: %d", ret)
	}
	return &Frame{pic: &pic}, nil
}

func (d *Decoder) Flush() {
	C.dav1d_flush(d.ctx)
}

func (d *Decoder) Close() {
	C.dav1d_close(&d.ctx)
}

func (d *Decoder) DecodeIVF(data []byte) ([]*Frame, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("слишком маленький файл для IVF")
	}
	if string(data[0:4]) != "DKIF" {
		return nil, fmt.Errorf("не IVF файл (магик не DKIF)")
	}
	data = data[32:]

	var frames []*Frame

	drain := func() error {
		for {
			frame, err := d.GetPicture()
			if err != nil {
				return err
			}
			if frame == nil {
				break
			}
			frames = append(frames, frame)
		}
		return nil
	}

	for len(data) >= 12 {
		frameSize := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
		if frameSize == 0 {
			data = data[12:]
			continue
		}
		if int(frameSize)+12 > len(data) {
			break
		}
		frameData := data[12 : 12+frameSize]

		if err := d.SendData(frameData); err != nil {
			return frames, fmt.Errorf("SendData: %w", err)
		}

		if err := drain(); err != nil {
			return frames, err
		}

		data = data[12+frameSize:]
	}

	if err := drain(); err != nil {
		return frames, err
	}

	return frames, nil
}

type Frame struct {
	pic *C.Dav1dPicture
}

func (f *Frame) Width() int  { return int(f.pic.p.w) }
func (f *Frame) Height() int { return int(f.pic.p.h) }

func (f *Frame) YCbCr() (y, u, v []byte, yStride, uvStride int) {
	ys := int(f.pic.stride[0])
	uvs := int(f.pic.stride[1])
	h := int(f.pic.p.h)

	subH := 1
	if f.pic.p.layout == C.DAV1D_PIXEL_LAYOUT_I420 {
		subH = 2
	}

	y = unsafe.Slice((*byte)(unsafe.Pointer(f.pic.data[0])), ys*h)
	u = unsafe.Slice((*byte)(unsafe.Pointer(f.pic.data[1])), uvs*(h/subH))
	v = unsafe.Slice((*byte)(unsafe.Pointer(f.pic.data[2])), uvs*(h/subH))
	yStride = ys
	uvStride = uvs
	return
}

func (f *Frame) Free() {
	C.dav1d_picture_unref(f.pic)
}
