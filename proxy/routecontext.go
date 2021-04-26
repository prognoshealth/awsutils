package proxy

import (
	"context"
	"encoding/base64"

	"github.com/aws/aws-lambda-go/events"
	"github.com/pkg/errors"
)

// RouteContext contains all the request information for a route when matched.
type RouteContext struct {
	Context context.Context
	Request events.APIGatewayV2HTTPRequest
	Params  map[string]string
}

// Body returns a string representation of the request body
func (ctx *RouteContext) Body() (string, error) {
	if ctx.Request.IsBase64Encoded {
		b, err := base64.StdEncoding.DecodeString(ctx.Request.Body)
		if err != nil {
			return "", errors.Wrapf(err, "unable to decode request body for request %v", ctx.Request)
		}

		return string(b), nil
	}

	return ctx.Request.Body, nil
}
