package builtin

import (
	"syscall"
	"unsafe"

	heap "github.com/apmckinlay/gsuneido/builtin/heapstack"
	. "github.com/apmckinlay/gsuneido/runtime"
	"github.com/apmckinlay/gsuneido/util/verify"
	"golang.org/x/sys/windows"
)

var kernel32 = windows.MustLoadDLL("kernel32.dll")

// dll Kernel32:GetComputerName(buffer lpBuffer, LONG* lpnSize) bool
var getComputerName = kernel32.MustFindProc("GetComputerNameA").Addr()
var _ = builtin0("GetComputerName()", func() Value {
	defer heap.FreeTo(heap.CurSize())
	const bufsize = 255
	buf := heap.Alloc(bufsize + 1)
	p := heap.Alloc(int32Size)
	pn := (*int32)(p)
	*pn = bufsize
	rtn, _, _ := syscall.Syscall(getComputerName, 2,
		uintptr(buf),
		uintptr(p),
		0)
	if rtn == 0 {
		return EmptyStr
	}
	verify.That(*pn <= bufsize)
	return bufToStr(buf, uintptr(*pn))
})

// dll Kernel32:GetModuleHandle(instring name) pointer
var getModuleHandle = kernel32.MustFindProc("GetModuleHandleA").Addr()
var _ = builtin1("GetModuleHandle(unused)",
	func(a Value) Value {
		rtn, _, _ := syscall.Syscall(getModuleHandle, 1,
			0,
			0, 0)
		return intRet(rtn)
	})

// dll Kernel32:GetLocaleInfo(long locale, long lctype, string lpLCData, long cchData) long
var getLocaleInfo = kernel32.MustFindProc("GetLocaleInfoA").Addr()
var _ = builtin2("GetLocaleInfo(a,b)",
	func(a, b Value) Value {
		defer heap.FreeTo(heap.CurSize())
		const bufsize = 255
		buf := heap.Alloc(bufsize + 1)
		syscall.Syscall6(getLocaleInfo, 4,
			intArg(a),
			intArg(b),
			uintptr(buf),
			uintptr(bufsize),
			0, 0)
		return bufToStr(buf, bufsize)
	})

// dll Kernel32:GetProcAddress(pointer hModule, instring procName) pointer
var getProcAddress = kernel32.MustFindProc("GetProcAddress").Addr()
var _ = builtin2("GetProcAddress(hModule, procName)",
	func(a, b Value) Value {
		defer heap.FreeTo(heap.CurSize())
		rtn, _, _ := syscall.Syscall(getProcAddress, 2,
			intArg(a),
			uintptr(stringArg(b)),
			0)
		return intRet(rtn)
	})

// dll Kernel32:GetProcessHeap() pointer
var getProcessHeap = kernel32.MustFindProc("GetProcessHeap").Addr()
var _ = builtin0("GetProcessHeap()",
	func() Value {
		rtn, _, _ := syscall.Syscall(getProcessHeap, 0, 0, 0, 0)
		return intRet(rtn)
	})

// dll Kernel32:GetVersionEx(OSVERSIONINFOEX* lpVersionInfo) bool
var getVersionEx = kernel32.MustFindProc("GetVersionExA").Addr()
var _ = builtin1("GetVersionEx(a)",
	func(a Value) Value {
		defer heap.FreeTo(heap.CurSize())
		p := heap.Alloc(nOSVERSIONINFOEX)
		ovi := (*OSVERSIONINFOEX)(p)
		ovi.dwOSVersionInfoSize = int32(nOSVERSIONINFOEX)
		rtn, _, _ := syscall.Syscall(getVersionEx, 1,
			uintptr(p),
			0, 0)
		a.Put(nil, SuStr("dwMajorVersion"),
			IntVal(int(ovi.dwMajorVersion)))
		a.Put(nil, SuStr("dwMinorVersion"),
			IntVal(int(ovi.dwMinorVersion)))
		a.Put(nil, SuStr("dwBuildNumber"),
			IntVal(int(ovi.dwBuildNumber)))
		a.Put(nil, SuStr("dwPlatformId"),
			IntVal(int(ovi.dwPlatformId)))
		a.Put(nil, SuStr("szCSDVersion"),
			strRet(ovi.szCSDVersion[:]))
		a.Put(nil, SuStr("wServicePackMajor"),
			IntVal(int(ovi.wServicePackMajor)))
		a.Put(nil, SuStr("wServicePackMinor"),
			IntVal(int(ovi.wServicePackMinor)))
		a.Put(nil, SuStr("wSuiteMask"),
			IntVal(int(ovi.wSuiteMask)))
		a.Put(nil, SuStr("wProductType"),
			IntVal(int(ovi.wProductType)))
		return boolRet(rtn)
	})

type OSVERSIONINFOEX struct {
	dwOSVersionInfoSize int32
	dwMajorVersion      int32
	dwMinorVersion      int32
	dwBuildNumber       int32
	dwPlatformId        int32
	szCSDVersion        [128]byte
	wServicePackMajor   int16
	wServicePackMinor   int16
	wSuiteMask          int16
	wProductType        byte
	wReserved           byte
}

const nOSVERSIONINFOEX = unsafe.Sizeof(OSVERSIONINFOEX{})

// dll Kernel32:GlobalAlloc(long flags, long size) pointer
var globalAlloc = kernel32.MustFindProc("GlobalAlloc").Addr()
var _ = builtin2("GlobalAlloc(flags, size)",
	func(a, b Value) Value {
		rtn, _, _ := syscall.Syscall(globalAlloc, 2,
			intArg(a),
			intArg(b),
			0)
		return intRet(rtn)
	})

// dll Kernel32:GlobalLock(pointer handle) pointer
var globalLock = kernel32.MustFindProc("GlobalLock").Addr()
var _ = builtin1("GlobalLock(hMem)",
	func(a Value) Value {
		rtn, _, _ := syscall.Syscall(globalLock, 1,
			intArg(a),
			0, 0)
		return intRet(rtn)
	})

// dll Kernel32:GlobalUnlock(pointer handle) bool
var globalUnlock = kernel32.MustFindProc("GlobalUnlock").Addr()
var _ = builtin1("GlobalUnlock(hMem)",
	func(a Value) Value {
		rtn, _, _ := syscall.Syscall(globalUnlock, 1,
			intArg(a),
			0, 0)
		return boolRet(rtn)
	})

// dll Kernel32:HeapAlloc(pointer hHeap, long dwFlags, long dwBytes) pointer
var heapAlloc = kernel32.MustFindProc("HeapAlloc").Addr()
var _ = builtin3("HeapAlloc(hHeap, dwFlags, dwBytes)",
	func(a, b, c Value) Value {
		rtn, _, _ := syscall.Syscall(heapAlloc, 3,
			intArg(a),
			intArg(b),
			intArg(c))
		return intRet(rtn)
	})

// dll Kernel32:HeapFree(pointer hHeap, long dwFlags, pointer lpMem) bool
var heapFree = kernel32.MustFindProc("HeapFree").Addr()
var _ = builtin3("HeapFree(hHeap, dwFlags, lpMem)",
	func(a, b, c Value) Value {
		rtn, _, _ := syscall.Syscall(heapFree, 3,
			intArg(a),
			intArg(b),
			intArg(c))
		return boolRet(rtn)
	})

var _ = builtin3("MulDiv(x, y, z)",
	func(a, b, c Value) Value {
		return IntVal(int(int64(ToInt(a)) * int64(ToInt(b)) / int64(ToInt(c))))
	})

var _ = builtin3("CopyMemory(destination, source, length)",
	func(a, b, c Value) Value {
		dst := uintptr(ToInt(a))
		src := ToStr(b)
		n := ToInt(c)
		for i := 0; i < n; i++ {
			*(*byte)(unsafe.Pointer(dst + uintptr(i))) = src[i]
		}
		return nil
	})

// dll bool Kernel32:CloseHandle(pointer handle)
var closeHandle = kernel32.MustFindProc("CloseHandle").Addr()
var _ = builtin1("CloseHandle(handle)",
	func(a Value) Value {
		rtn, _, _ := syscall.Syscall(closeHandle, 1,
			intArg(a),
			0, 0)
		return boolRet(rtn)
	})

// dll bool Kernel32:CopyFile(string from, string to, bool failIfExists)
var copyFile = kernel32.MustFindProc("CopyFileA").Addr()
var _ = builtin3("CopyFile(from, to, failIfExists)",
	func(a, b, c Value) Value {
		defer heap.FreeTo(heap.CurSize())
		rtn, _, _ := syscall.Syscall(copyFile, 3,
			uintptr(stringArg(a)),
			uintptr(stringArg(b)),
			boolArg(c))
		return boolRet(rtn)
	})

// dll bool Kernel32:FindClose(pointer hFindFile)
var findClose = kernel32.MustFindProc("FindClose").Addr()
var _ = builtin1("FindClose(hFindFile)",
	func(a Value) Value {
		rtn, _, _ := syscall.Syscall(findClose, 1,
			intArg(a),
			0, 0)
		return boolRet(rtn)
	})

// dll bool Kernel32:FlushFileBuffers(handle hFile)
var flushFileBuffers = kernel32.MustFindProc("FlushFileBuffers").Addr()
var _ = builtin1("FlushFileBuffers(hFile)",
	func(a Value) Value {
		rtn, _, _ := syscall.Syscall(flushFileBuffers, 1,
			intArg(a),
			0, 0)
		return boolRet(rtn)
	})

// dll pointer Kernel32:GetCurrentProcess()
var getCurrentProcess = kernel32.MustFindProc("GetCurrentProcess").Addr()
var _ = builtin0("GetCurrentProcess()",
	func() Value {
		rtn, _, _ := syscall.Syscall(getCurrentProcess, 0, 0, 0, 0)
		return intRet(rtn)
	})

// dll long Kernel32:GetCurrentProcessId()
var getCurrentProcessId = kernel32.MustFindProc("GetCurrentProcessId").Addr()
var _ = builtin0("GetCurrentProcessId()",
	func() Value {
		rtn, _, _ := syscall.Syscall(getCurrentProcessId, 0, 0, 0, 0)
		return intRet(rtn)
	})

// dll long Kernel32:GetCurrentThreadId()
var getCurrentThreadId = kernel32.MustFindProc("GetCurrentThreadId").Addr()
var _ = builtin0("GetCurrentThreadId()",
	func() Value {
		rtn, _, _ := syscall.Syscall(getCurrentThreadId, 0, 0, 0, 0)
		return intRet(rtn)
	})

// dll long Kernel32:GetFileAttributes(
// 		[in] string lpFileName)
var getFileAttributes = kernel32.MustFindProc("GetFileAttributesA").Addr()
var _ = builtin1("GetFileAttributes(lpFileName)",
	func(a Value) Value {
		defer heap.FreeTo(heap.CurSize())
		rtn, _, _ := syscall.Syscall(getFileAttributes, 1,
			uintptr(stringArg(a)),
			0, 0)
		return intRet(rtn)
	})

// dll long Kernel32:GetLastError()
var getLastError = kernel32.MustFindProc("GetLastError").Addr()
var _ = builtin0("GetLastError()",
	func() Value {
		rtn, _, _ := syscall.Syscall(getLastError, 0, 0, 0, 0)
		return intRet(rtn)
	})

// dll pointer Kernel32:GetStdHandle(long nStdHandle)
var getStdHandle = kernel32.MustFindProc("GetStdHandle").Addr()
var _ = builtin1("GetStdHandle(nStdHandle)",
	func(a Value) Value {
		rtn, _, _ := syscall.Syscall(getStdHandle, 1,
			intArg(a),
			0, 0)
		return intRet(rtn)
	})

// dll int64 Kernel32:GetTickCount64()
var getTickCount64 = kernel32.MustFindProc("GetTickCount64").Addr()
var _ = builtin0("GetTickCount()",
	func() Value {
		rtn, _, _ := syscall.Syscall(getTickCount64, 0, 0, 0, 0)
		return intRet(rtn)
	})

// dll long Kernel32:GetWindowsDirectory(string lpBuffer, long size)
var getWindowsDirectory = kernel32.MustFindProc("GetWindowsDirectoryA").Addr()
var _ = builtin0("GetWindowsDirectory()",
	func() Value {
		defer heap.FreeTo(heap.CurSize())
		const bufsize = 256
		buf := heap.Alloc(bufsize + 1)
		syscall.Syscall(getWindowsDirectory, 2,
			uintptr(buf),
			uintptr(bufsize),
			0)
		return bufToStr(buf, bufsize)
	})

// dll pointer Kernel32:GlobalFree(pointer hglb)
var globalFree = kernel32.MustFindProc("GlobalFree").Addr()
var _ = builtin1("GlobalFree(hglb)",
	func(a Value) Value {
		rtn, _, _ := syscall.Syscall(globalFree, 1,
			intArg(a),
			0, 0)
		return intRet(rtn)
	})

// dll long Kernel32:GlobalSize(pointer handle)
var globalSize = kernel32.MustFindProc("GlobalSize").Addr()
var _ = builtin1("GlobalSize(handle)",
	func(a Value) Value {
		rtn, _, _ := syscall.Syscall(globalSize, 1,
			intArg(a),
			0, 0)
		return intRet(rtn)
	})

// dll pointer Kernel32:LoadLibrary([in] string library)
var loadLibrary = kernel32.MustFindProc("LoadLibraryA").Addr()
var _ = builtin1("LoadLibrary(library)",
	func(a Value) Value {
		defer heap.FreeTo(heap.CurSize())
		rtn, _, _ := syscall.Syscall(loadLibrary, 1,
			uintptr(stringArg(a)),
			0, 0)
		return intRet(rtn)
	})

// dll pointer Kernel32:LoadResource(pointer module, pointer res)
var loadResource = kernel32.MustFindProc("LoadResource").Addr()
var _ = builtin2("LoadResource(module, res)",
	func(a, b Value) Value {
		rtn, _, _ := syscall.Syscall(loadResource, 2,
			intArg(a),
			intArg(b),
			0)
		return intRet(rtn)
	})

// dll bool Kernel32:SetCurrentDirectory(string lpPathName)
var setCurrentDirectory = kernel32.MustFindProc("SetCurrentDirectoryA").Addr()
var _ = builtin1("SetCurrentDirectory(lpPathName)",
	func(a Value) Value {
		defer heap.FreeTo(heap.CurSize())
		rtn, _, _ := syscall.Syscall(setCurrentDirectory, 1,
			uintptr(stringArg(a)),
			0, 0)
		return boolRet(rtn)
	})

// dll bool Kernel32:SetFileAttributes(
//		[in] string lpFileName, long dwFileAttributes)
var setFileAttributes = kernel32.MustFindProc("SetFileAttributesA").Addr()
var _ = builtin2("SetFileAttributes(lpFileName, dwFileAttributes)",
	func(a, b Value) Value {
		defer heap.FreeTo(heap.CurSize())
		rtn, _, _ := syscall.Syscall(setFileAttributes, 2,
			uintptr(stringArg(a)),
			intArg(b),
			0)
		return boolRet(rtn)
	})

// dll handle Kernel32:CreateFile([in] string lpFileName, long dwDesiredAccess,
//		long dwShareMode, SECURITY_ATTRIBUTES* lpSecurityAttributes,
//		long dwCreationDistribution, long dwFlagsAndAttributes,
//		pointer hTemplateFile)
var createFile = kernel32.MustFindProc("CreateFileA").Addr()
var _ = builtin7("CreateFile(lpFileName, dwDesiredAccess, dwShareMode,"+
	"lpSecurityAttributes, dwCreationDistribution, dwFlagsAndAttributes,"+
	"hTemplateFile)",
	func(a, b, c, d, e, f, g Value) Value {
		defer heap.FreeTo(heap.CurSize())
		p := heap.Alloc(nSECURITY_ATTRIBUTES)
		*(*SECURITY_ATTRIBUTES)(p) = SECURITY_ATTRIBUTES{
			nLength:              int32(nSECURITY_ATTRIBUTES),
			lpSecurityDescriptor: getHandle(d, "lpSecurityDescriptor"),
			bInheritHandle:       getBool(d, "bInheritHandle"),
		}
		rtn, _, _ := syscall.Syscall9(createFile, 7,
			uintptr(stringArg(a)),
			intArg(b),
			intArg(c),
			uintptr(p),
			intArg(e),
			intArg(f),
			intArg(g),
			0, 0)
		return intRet(rtn)
	})

type SECURITY_ATTRIBUTES struct {
	nLength              int32
	lpSecurityDescriptor HANDLE
	bInheritHandle       BOOL
}

const nSECURITY_ATTRIBUTES = unsafe.Sizeof(SECURITY_ATTRIBUTES{})

// dll long Kernel32:GetFileSize(handle hf, LONG* hiword)
var getFileSize = kernel32.MustFindProc("GetFileSize").Addr()
var _ = builtin2("GetFileSize(a, b/*unused*/)",
	func(a, b Value) Value {
		rtn, _, _ := syscall.Syscall(getFileSize, 2,
			intArg(a),
			0,
			0)
		return intRet(rtn)
	})

// dll bool Kernel32:GetVolumeInformation([in] string lpRootPathName,
//		string lpVolumeNameBuffer, long nVolumeNameSize, LONG* lpVolumeSerialNumber,
//		LONG* lpMaximumComponentLength, LONG* lpFileSystemFlags,
//		string lpFileSystemNameBuffer, long nFileSystemNameSize)
var getVolumeInformation = kernel32.MustFindProc("GetVolumeInformationA").Addr()
var _ = builtin1("GetVolumeName(vol = 'c:\\\\')",
	func(a Value) Value {
		defer heap.FreeTo(heap.CurSize())
		const bufsize = 255
		buf := heap.Alloc(bufsize + 1)
		rtn, _, _ := syscall.Syscall9(getVolumeInformation, 8,
			uintptr(stringArg(a)),
			uintptr(buf),
			uintptr(bufsize),
			0,
			0,
			0,
			0,
			0,
			0)
		if rtn == 0 {
			return EmptyStr
		}
		return bufToStr(buf, bufsize)
	})

type MEMORYSTATUSEX struct {
	dwLength     uint32
	dwMemoryLoad uint32
	ullTotalPhys uint64
	unused       [6]uint64
}

const nMEMORYSTATUSEX = unsafe.Sizeof(MEMORYSTATUSEX{})

var globalMemoryStatusEx = kernel32.MustFindProc("GlobalMemoryStatusEx").Addr()

var _ = builtin0("SystemMemory()", func() Value {
	defer heap.FreeTo(heap.CurSize())
	p := heap.Alloc(nMEMORYSTATUSEX)
	r, _, _ := syscall.Syscall(globalMemoryStatusEx, 1,
		uintptr(p),
		0, 0)
	if r == 0 {
		return Zero
	}
	return Int64Val(int64((*MEMORYSTATUSEX)(p).ullTotalPhys))
})

var _ = builtin0("OperatingSystem()", func() Value {
	return SuStr("Windows") //TODO version
})

// dll bool Kernel32:GetDiskFreeSpaceEx(
// 	[in] string			directoryName,
// 	ULARGE_INTEGER*		freeBytesAvailableToCaller,
// 	ULARGE_INTEGER*		totalNumberOfBytes,
// 	ULARGE_INTEGER*		totalNumberOfFreeBytes
// 	)
var getDiskFreeSpaceEx = kernel32.MustFindProc("GetDiskFreeSpaceExA").Addr()

var _ = builtin1("GetDiskFreeSpace(dir = '.')", func(arg Value) Value {
	defer heap.FreeTo(heap.CurSize())
	p := heap.Alloc(int64Size)
	syscall.Syscall6(getDiskFreeSpaceEx, 4,
		uintptr(stringArg(arg)),
		uintptr(p),
		0,
		0,
		0, 0)
	return Int64Val(*(*int64)(p))
})
