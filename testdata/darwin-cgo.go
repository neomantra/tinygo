package main

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func checkMode(path string, want os.FileMode) {
	info, err := os.Stat(path)
	check(err)
	if info.Mode().Perm() != want {
		panic("incorrect file mode")
	}
}

func main() {
	dir, err := os.MkdirTemp("", "tinygo-darwin-cgo-")
	check(err)
	defer os.RemoveAll(dir)
	umask := unix.Umask(0)
	defer unix.Umask(umask)

	fd, err := unix.Open(dir+"/open", unix.O_CREAT|unix.O_RDWR, 0640)
	check(err)
	defer unix.Close(fd)
	checkMode(dir+"/open", 0640)
	println("unix.Open passed")

	dirfd, err := unix.Open(dir, unix.O_RDONLY, 0)
	check(err)
	defer unix.Close(dirfd)
	fdAt, err := unix.Openat(dirfd, "openat", unix.O_CREAT|unix.O_RDWR, 0600)
	check(err)
	defer unix.Close(fdAt)
	checkMode(dir+"/openat", 0600)
	println("unix.Openat passed")

	dup, err := unix.FcntlInt(uintptr(fd), unix.F_DUPFD, 50)
	check(err)
	defer unix.Close(dup)
	if dup < 50 {
		panic("incorrect duplicate descriptor")
	}
	println("unix.FcntlInt passed")

	lock := unix.Flock_t{Type: unix.F_WRLCK, Whence: unix.SEEK_SET}
	check(unix.FcntlFlock(uintptr(fd), unix.F_GETLK, &lock))
	if lock.Type != unix.F_UNLCK {
		panic("incorrect lock type")
	}
	println("unix.FcntlFlock passed")

	var pipe [2]int
	check(unix.Pipe(pipe[:]))
	defer unix.Close(pipe[0])
	defer unix.Close(pipe[1])
	n, err := unix.Write(pipe[1], []byte("hello"))
	check(err)
	if n != 5 {
		panic("incomplete pipe write")
	}
	// FIONREAD is _IOR('f', 127, int).
	// See https://github.com/apple-oss-distributions/xnu/blob/main/bsd/sys/filio.h.
	const fionread = 0x4004667f
	available, err := unix.IoctlGetInt(pipe[0], fionread)
	check(err)
	if available != n {
		panic("incorrect available byte count")
	}
	println("unix.IoctlGetInt passed")

	stdlibFD, err := syscall.Open(dir+"/stdlib", syscall.O_CREAT|syscall.O_RDWR, 0604)
	check(err)
	defer syscall.Close(stdlibFD)
	checkMode(dir+"/stdlib", 0604)
	println("syscall.Open passed")

	before, err := unix.FcntlInt(uintptr(pipe[0]), unix.F_GETFL, 0)
	check(err)
	check(syscall.SetNonblock(pipe[0], true))
	after, err := unix.FcntlInt(uintptr(pipe[0]), unix.F_GETFL, 0)
	check(err)
	if after != before|unix.O_NONBLOCK {
		panic("incorrect nonblocking flags")
	}
	check(syscall.SetNonblock(pipe[0], false))
	after, err = unix.FcntlInt(uintptr(pipe[0]), unix.F_GETFL, 0)
	check(err)
	if after != before {
		panic("file flags were not restored")
	}
	println("syscall.SetNonblock passed")
}
