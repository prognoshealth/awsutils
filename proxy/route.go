package proxy

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/pkg/errors"
)

// RouteHandler defines the function interface the route uses to execute a
// request when the route is matched.
type RouteHandler func(*RouteContext) (events.APIGatewayProxyResponse, error)

// Route defines a HttpMethod and Regex that are used in combination for
// matching against an incoming request. When a match occurs the configured
// handler is called.
type Route struct {
	Method  HttpMethod
	Regex   *regexp.Regexp
	Handler RouteHandler
}

// NewRoute returns a Route for the specified method, pattern and handler.
func NewRoute(method HttpMethod, pattern string, handler RouteHandler) (*Route, error) {
	rx, err := regexp.Compile("^" + pattern + "/?$")

	if err != nil {
		return nil, errors.Wrapf(err, "failed compiling regex pattern '%s'", pattern)
	}

	route := &Route{
		Method:  method,
		Regex:   rx,
		Handler: handler,
	}

	return route, nil
}

// String returns a string representation of this route.
func (route *Route) String() string {
	return fmt.Sprintf("%s %s", route.Method, route.Regex)
}

// IsMatch return true if there is a match otherwise false. The match groups are
// also returned.
func (route *Route) IsMatch(request events.APIGatewayV2HTTPRequest) (bool, []string) {
	if route.Method.String() != request.RequestContext.HTTP.Method {
		return false, nil
	}

	groups := route.Regex.FindStringSubmatch(request.RawPath)

	if len(groups) == 0 {
		return false, nil
	}

	return true, groups
}

// extractParamsFromPath pulls the paramenters set on the aws api gateway path
// parameters.
func (route *Route) extractParamsFromPath(params map[string]string, request events.APIGatewayV2HTTPRequest) {
	for k, v := range request.PathParameters {
		params[k] = v
	}
}

// extractParamsFromQueryString pulls the params from the query parameters.
func (route *Route) extractParamsFromQueryString(params map[string]string, request events.APIGatewayV2HTTPRequest) {
	for k, v := range request.QueryStringParameters {
		params[k] = v
	}
}

// extractParamsFromURIRegex pulls the named groups values matched via the regex
// into the provided params map.
func (route *Route) extractParamsFromURIRegex(params map[string]string, groups []string) {
	for i, name := range route.Regex.SubexpNames() {
		if i != 0 && name != "" && groups[i] != "" {
			params[name] = groups[i]
		}
	}
}

// extractParamsFromFormPost extracts the params from a POSTed body with content
// type 'application/x-www-form-urlencoded'.
func (route *Route) extractParamsFromFormPost(params map[string]string, request events.APIGatewayV2HTTPRequest) error {
	if POST.String() != request.RequestContext.HTTP.Method {
		return nil
	}

	if request.Headers["content-type"] != "application/x-www-form-urlencoded" {
		return nil
	}

	body := ""

	if request.IsBase64Encoded {
		b, err := base64.StdEncoding.DecodeString(request.Body)
		if err != nil {
			return errors.Wrapf(err, "unable to decode request form params %v", request)
		}

		body = string(b)
	} else {
		body = request.Body
	}

	kvs := strings.Split(body, "&")

	for _, kv := range kvs {
		kvSplit := strings.Split(kv, "=")

		if len(kvSplit) != 2 {
			return fmt.Errorf("invalid key/value pair in form post for %v", request)
		}

		v, err := url.QueryUnescape(kvSplit[1])
		if err != nil {
			return errors.Wrapf(err, "unable to decode value '%v'", kvSplit[1])
		}

		params[kvSplit[0]] = v
	}

	return nil
}

// Context constructs a RouteContext for the route for passing to the handler.
// The 'Params' that get set on the context are extracted from the request with
// the following precedence:
//
//	1) Form POSTs
//  2) Route defined regex capture
//  3) Query string
//  4) AWS API Gateway configured PathParameters.
func (route *Route) Context(ctx context.Context, request events.APIGatewayV2HTTPRequest, groups []string) (*RouteContext, error) {
	if len(groups) == 0 {
		return nil, fmt.Errorf("No matches available, unabled to generate context for route %v", route)
	}

	params := make(map[string]string)
	route.extractParamsFromPath(params, request)
	route.extractParamsFromQueryString(params, request)
	route.extractParamsFromURIRegex(params, groups)
	err := route.extractParamsFromFormPost(params, request)

	if err != nil {
		return nil, errors.Wrapf(err, "failed extractParamsFromFormPost")
	}

	return &RouteContext{
		Context: ctx,
		Request: request,
		Params:  params,
	}, nil
}

// Follow extracts the route context for the given request and executed the
// route's handler function.
func (route *Route) Follow(ctx context.Context, request events.APIGatewayV2HTTPRequest, groups []string) (events.APIGatewayProxyResponse, error) {
	rctx, err := route.Context(ctx, request, groups)

	if err != nil {
		return events.APIGatewayProxyResponse{}, errors.Wrapf(err, "failed getting context for route %v", route.Regex)
	}

	return route.Handler(rctx)
}
