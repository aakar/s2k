package common

import (
	"unicode/utf16"
	"unsafe"
)

// In some cases we cannot use platform specific syscall or windows functions

// UTF16PtrToUTF16 copies WSTR content into slice with terminating NUL character
func UTF16PtrToUTF16(p *uint16) []uint16 {
	end, n := unsafe.Pointer(p), 0
	for *(*uint16)(end) != 0 {
		end = unsafe.Pointer(uintptr(end) + unsafe.Sizeof(*p))
		n++
	}
	res := make([]uint16, n+1) // make sure there is place to terminating zero
	copy(res, unsafe.Slice(p, n))
	return res
}

// UTF16ToString converts zero-terminated UTF-16 string (LPWSTR) inside []utf16 slice to Go string
// NOTE: slice size may be bigger that string length
func UTF16ToString(p []uint16) string {
	length := len(p)
	if length == 0 {
		return ""
	}

	symSize := unsafe.Sizeof(p[0])
	str := make([]uint16, length)

	ptr := unsafe.Pointer(&p[0])
	for i := 0; i < int(length); i++ {
		sym := *(*uint16)(ptr)
		if sym == 0 {
			str = str[:i]
			break
		}
		str[i] = sym
		ptr = unsafe.Add(ptr, symSize)
	}
	return string(utf16.Decode(str))
}
