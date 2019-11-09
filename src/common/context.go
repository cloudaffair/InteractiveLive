package common

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"context"
	"time"
)

const ctxGinContextKey     = "ginContextKey"
const ctxServiceKey        = "service"
const serviceName = "ILA"

func GinContext(ctx context.Context) (*gin.Context, error) {
	if ctx != nil {
		// ctx.Value returns nil if ctx has no value for the key;
		if cgin, ok := ctx.Value(ctxGinContextKey).(*gin.Context); ok {
			return cgin, nil
		}
	}
	return nil, fmt.Errorf("no Gin context in context")
}

func GoogleContext(cgin *gin.Context) (context.Context, error) {
	ctx, ok := cgin.Keys["context"]
	fmt.Println(ctx, ok)
	if ok {
		return ctx.(context.Context), nil
	}
	return nil, NewError("No Google context found in Gin context")
}

func NewContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	var ctx context.Context
	var cancel context.CancelFunc

	if timeout > 0 {
		// The request has a timeout, so create a context that is
		// canceled automatically when the timeout expires.
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}

	ctx = SetService(ctx, serviceName)

	return ctx, cancel
}

func SetService(ctx context.Context, service string) context.Context {
	return context.WithValue(ctx, ctxServiceKey, service)
}

func CreateLoadGoogleContext(cgin *gin.Context) context.Context {
	ctx, _ := NewContext(0)
	CheckContext(ctx)
	if cgin == nil {
		return ctx
	}
	// set gin context in our context and vice versa
	ctx = SetGinContext(ctx, cgin)
	SetGoogleContext(cgin, ctx)

	return ctx
}

// this set the Gin context within the Google context
func SetGinContext(ctx context.Context, cgin *gin.Context) context.Context {
	return context.WithValue(ctx, ctxGinContextKey, cgin)
}

// this sets the Google context within the Gin context
func SetGoogleContext(cgin *gin.Context, ctx context.Context) {
	cgin.Set("context", ctx)
}

func CheckContext(ctx context.Context) {
	if ctx == nil {
		fmt.Errorf("nil context")
	}
}
