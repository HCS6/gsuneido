package builtin

import (
	"unsafe"

	. "github.com/apmckinlay/gsuneido/runtime"
	"golang.org/x/sys/windows"
)

var ole32 = windows.NewLazyDLL("ole32.dll")

// dll long Ole32:CreateStreamOnHGlobal(
// 	pointer hGlobal,
// 	bool fDeleteOnRelease,
// 	POINTER* ppstm)
var createStreamOnHGlobal = ole32.NewProc("CreateStreamOnHGlobal")
var _ = builtin3("CreateStreamOnHGlobal(hGlobal, fDeleteOnRelease, ppstm)",
	func(a, b, c Value) Value {
		var p uintptr
		rtn, _, _ := createStreamOnHGlobal.Call(
			intArg(a),
			boolArg(b),
			uintptr(unsafe.Pointer(&p)))
		c.Put(nil, SuStr("x"), IntVal(int(p)))
		return intRet(rtn)
	})

var oleaut32 = windows.NewLazyDLL("oleaut32.dll")

// dll long OleAut32:OleLoadPicture(
//		pointer lpstream,
//		long lSize,
//		bool fRunmode,
//		GUID* riid,
//		POINTER* lplpvObj)
var oleLoadPicture = oleaut32.NewProc("OleLoadPicture")
var _ = builtin5("OleLoadPicture(lpstream, lSize, fRunmode, riid, lplpvobj)",
	func(a, b, c, d, e Value) Value {
		var p uintptr
		guid := GUID{
			Data1: getInt32(d, "Data1"),
			Data2: int16(getInt(d, "Data2")),
			Data3: int16(getInt(d, "Data3")),
		}
		data4 := d.Get(nil, SuStr("Data4"))
		for i := 0; i < 8; i++ {
			guid.Data4[i] = byte(ToInt(data4.Get(nil, SuInt(i))))
		}
		rtn, _, _ := oleLoadPicture.Call(
			intArg(a),
			intArg(b),
			boolArg(c),
			uintptr(unsafe.Pointer(&guid)),
			uintptr(unsafe.Pointer(&p)))
		e.Put(nil, SuStr("x"), IntVal(int(p)))
		return intRet(rtn)
	})

type GUID struct {
	Data1 int32
	Data2 int16
	Data3 int16
	Data4 [8]byte
}