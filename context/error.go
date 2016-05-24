package context

import (
	"github.com/danielkrainas/canaria-api/api/errcode"
)

func WithErrors(ctx Context, errors errcode.Errors) Context {
	return WithValue(ctx, "errors", errors)
}

func AppendError(ctx Context, ec errcode.ErrorCode) Context {
	errors := GetErrors(ctx)
	errors = append(errors, ec)
	return WithErrors(ctx, errors)
}

func GetErrors(ctx Context) errcode.Errors {
	if errors, ok := ctx.Value("errors").(errcode.Errors); errors != nil && ok {
		return errors
	}

	return make(errcode.Errors, 0)
}
