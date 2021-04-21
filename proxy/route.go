package proxy

import (
	"context"
	"fmt"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"github.com/pkg/errors"
)

// RouteHandler defines the function interface the route uses to execute a
// request when the route is matched.
type RouteHandler func(*RouteContext) (events.APIGatewayProxyResponse, error)

// RouteContext contains all the request information for a route when matched.
type RouteContext struct {
	Context context.Context
	Request events.APIGatewayV2HTTPRequest
	Params  map[string]string
}

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

// Context constructs a RouteContext for the route for passing to the handler.
func (route *Route) Context(ctx context.Context, request events.APIGatewayV2HTTPRequest, groups []string) (*RouteContext, error) {
	if len(groups) == 0 {
		return nil, fmt.Errorf("No matches available, unabled to generate context for route %v", route)
	}

	namedGroups := make(map[string]string)
	for i, name := range route.Regex.SubexpNames() {
		if i != 0 && name != "" && groups[i] != "" {
			namedGroups[name] = groups[i]
		}
	}

	return &RouteContext{
		Context: ctx,
		Request: request,
		Params:  namedGroups,
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
