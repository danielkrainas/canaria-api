package context

import (
	"github.com/danielkrainas/canaria-api/common"
)

func WithCanary(ctx Context, c *common.Canary) Context {
	return WithValue(ctx, "canary", c)
}

func GetCanary(ctx Context) *common.Canary {
	if c, ok := ctx.Value("canary").(*common.Canary); c != nil && ok {
		return c
	}

	return nil
}

func GetCanaryID(ctx Context) string {
	return GetStringValue(ctx, "vars.canary_id")
}
