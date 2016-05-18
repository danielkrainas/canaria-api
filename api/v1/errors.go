package v1

import (
	"net/http"

	"github.com/danielkrainas/canaria-api/api/errcode"
)

const errGroup = "canary.api.v1"

var (
	ErrorCodeTTLInvalid = errcode.Register(errGroup, errcode.ErrorDescriptor{
		Value:          "TTL_INVALID",
		Message:        "",
		Description:    "",
		HttpStatusCode: http.StatusBadRequest,
	})

	ErrorCodeCanaryUnknown = errcode.Register(errGroup, errcode.ErrorDescriptor{
		Value:          "CANARY_UNKNOWN",
		Message:        "",
		Description:    "",
		HttpStatusCode: http.StatusNotFound,
	})

	ErrorCodeCanaryDead = errcode.Register(errGroup, errcode.ErrorDescriptor{
		Value:          "CANARY_DEAD",
		Message:        "",
		Description:    "",
		HttpStatusCode: http.StatusNotFound,
	})
)
