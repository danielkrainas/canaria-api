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

	ErrorCodeCanaryInvalid = errcode.Register(errGroup, errcode.ErrorDescriptor{
		Value:          "CANARY_INVALID",
		Message:        "",
		Description:    "",
		HttpStatusCode: http.StatusBadRequest,
	})

	ErrorCodeWebhookSetupInvalid = errcode.Register(errGroup, errcode.ErrorDescriptor{
		Value:          "WEBHOOK_INVALID",
		Message:        "",
		Description:    "",
		HttpStatusCode: http.StatusBadRequest,
	})

	ErrorCodeWebhookUnknown = errcode.Register(errGroup, errcode.ErrorDescriptor{
		Value:          "WEBHOOK_UNKNOWN",
		Message:        "",
		Description:    "",
		HttpStatusCode: http.StatusNotFound,
	})

	ErrorCodeWebhookFailed = errcode.Register(errGroup, errcode.ErrorDescriptor{
		Value:          "WEBHOOK_FAILED",
		Message:        "",
		Description:    "",
		HttpStatusCode: http.StatusAccepted,
	})
)
