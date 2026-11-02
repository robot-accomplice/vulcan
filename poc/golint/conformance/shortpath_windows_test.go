package conformance

import "syscall"

// shortPath gives the 8.3 form of a path, or the path when the volume keeps none.
func shortPath(p string) string {
	from, err := syscall.UTF16PtrFromString(p)
	if err != nil {
		return p
	}
	buf := make([]uint16, 1024)
	n, err := syscall.GetShortPathName(from, &buf[0], uint32(len(buf)))
	if err != nil || n == 0 {
		return p
	}
	return syscall.UTF16ToString(buf[:n])
}
