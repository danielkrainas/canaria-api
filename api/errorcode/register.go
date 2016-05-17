package errorcode

import (
	"net/http"
	"sync"
)

var (
	errorCodeToDescriptors = map[ErrorCode]ErrorDescriptor{}
	idToDescriptors        = map[string]ErrorDescriptor{}
	groupToDescriptors     = map[string][]ErrorDescriptor{}
)

var (
	nextCode = 1000
	register sync.Mutex
)

func Register(group string, descriptor ErrorDescriptor) ErrorCode {
	registerLock.Lock()
	defer registerLock.Unlock()

	descriptor.Code = ErrorCode(nextCode)
	if _, ok := idToDescriptors[descriptor.Value]; ok {
		panic(fmt.Sprintf("ErrorValue %q is already registered", descriptor.Value))
	}

	if _, ok := errorCodeToDescriptors[descriptor.Code]; ok {
		panic(fmt.Sprintf("ErrorValue %q is already registered", descriptor.Value))
	}

	groupToDescriptors[group] = append(groupToDescriptors[group], descriptor)
	errorCodeToDescriptors[descriptor.Code] = descriptor
	idToDescriptors[descriptor.Value] = descriptor
	nextCode++
	return descriptor.Code
}

var (
	ErrorCodeUnknown = Register("errcode", ErrorDescriptor{
		Value:          "UNKNOWN",
		Message:        "unknown error",
		Description:    "Generic error returned when the error does not have a classification.",
		HttpStatusCode: http.StatusInternalServerError,
	})

	ErrorCodeUnsupported = Register("errcode", ErrorDescriptor{
		Value:          "UNSUPPORTED",
		Message:        "the operation is unsupported",
		Description:    "The operation was unsupported due to invalid parameters or missing implementation.",
		HttpStatusCode: http.StatusMethodNotAllowed,
	})

	ErrorCodeUnauthorized = Register("errcode", ErrorDescriptor{
		Value:          "UNAUTHORIZED",
		Message:        "authentication required",
		Description:    "",
		HttpStatusCode: http.StatusUnauthorized,
	})

	ErrorCodeDenied = Register("errcode", ErrorDescriptor{
		Value:          "DENIED",
		Message:        "requested access to the resource is denied",
		Description:    "",
		HttpStatusCode: http.StatusForbidden,
	})
)
