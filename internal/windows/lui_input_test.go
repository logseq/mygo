//go:build windows && (amd64 || arm64)

package windows

import (
	"reflect"
	"runtime"
	"testing"
	"unicode/utf16"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

func TestLUIIMERequestDocumentFeedBufferBoundaries(t *testing.T) {
	for _, request := range []uintptr{imrDocumentFeed, imrReconvertString} {
		for _, text := range []string{"", "A🙂中"} {
			t.Run(text, func(t *testing.T) {
				input := platform.TextInputState{Active: true, Text: text}
				if text != "" {
					input.Start, input.End = 1, 2
				}
				s := surface{input: input}
				head := uint32(unsafe.Sizeof(reconvertString{}))
				units := utf16.Encode([]rune(text))
				size := uintptr(head) + uintptr(2*len(units))
				if got := s.imeRequest(request, 0); got != size {
					t.Fatalf("request %d: buffer size %d, want %d", request, got, size)
				}
				buf := make([]uint32, (size+3)/4)
				rs := (*reconvertString)(unsafe.Pointer(&buf[0]))
				rs.Size = uint32(size)
				if got := s.imeRequest(request, uintptr(unsafe.Pointer(rs))); got != size {
					t.Fatalf("request %d: wrote %d bytes, want %d", request, got, size)
				}
				wantStart, wantLength := uint32(0), uint32(0)
				if text != "" {
					wantStart, wantLength = 2, 2
				}
				want := reconvertString{Size: uint32(size), StrLen: uint32(len(units)), StrOffset: head,
					CompStrOffset: wantStart, CompStrLen: wantLength, TargetStrOffset: wantStart, TargetStrLen: wantLength}
				if *rs != want {
					t.Fatalf("request %d: header %+v, want %+v", request, *rs, want)
				}
				if len(units) != 0 {
					got := unsafe.Slice((*uint16)(unsafe.Add(unsafe.Pointer(rs), head)), len(units))
					if !reflect.DeepEqual(got, units) {
						t.Fatalf("request %d: UTF-16 %v, want %v", request, got, units)
					}
				}
				runtime.KeepAlive(buf)
				rs.Size = uint32(size - 1)
				before := *rs
				if got := s.imeRequest(request, uintptr(unsafe.Pointer(rs))); got != 0 || *rs != before {
					t.Fatalf("request %d: undersized buffer was accepted or modified", request)
				}
				runtime.KeepAlive(buf)
			})
		}
	}
}
