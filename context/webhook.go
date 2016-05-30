package context

import (
	"github.com/danielkrainas/canaria-api/common"
)

func WithCanaryHook(ctx Context, h *common.WebHook) Context {
	return WithValue(ctx, "webhook", h)
}

func GetCanaryHookID(ctx Context) string {
	return GetStringValue(ctx, "vars.hook_id")
}

func GetCanaryHook(ctx Context) *common.WebHook {
	if webhook, ok := ctx.Value("webhook").(*common.WebHook); ok {
		return webhook
	}

	return nil
}
