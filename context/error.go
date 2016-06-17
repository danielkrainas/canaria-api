package context

import (
	"github.com/danielkrainas/canaria-api/api/errcode"
)

type errorsContext struct {
	Context
	errors errcode.Errors
}

func (ctx *errorsContext) Value(key interface{}) interface{} {
	if ks, ok := key.(string); ok {
		if ks == "errors" {
			return ctx.errors
		} else if ks == "errors.ctx" {
			return ctx
		}
	}

	return ctx.Context.Value(key)
}

func WithErrors(ctx Context, errors errcode.Errors) Context {
	ectx := getErrorContext(ctx)
	if ectx == nil {
		ectx = &errorsContext{
			Context: ctx,
			errors:  errors,
		}

		ctx = ectx
	} else {
		ectx.errors = errors
	}

	return ctx
}

func AppendError(ctx Context, err error) Context {
	ectx := getErrorContext(ctx)
	if ectx == nil {
		return WithErrors(ctx, errcode.Errors{err})
	}

	ectx.errors = append(ectx.errors, err)
	return ctx
}

func GetErrors(ctx Context) errcode.Errors {
	if errors, ok := ctx.Value("errors").(errcode.Errors); errors != nil && ok {
		return errors
	}

	return make(errcode.Errors, 0)
}

func getErrorContext(ctx Context) *errorsContext {
	v := ctx.Value("errors.ctx")
	if v == nil {
		return nil
	} else if ectx, ok := v.(*errorsContext); ok {
		return ectx
	}

	return nil
}
