package proxy

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/pkg/errors"
)

// ErrorHandler defines the function interface the router uses to handle any
// error that occurs while processing routes.
type ErrorHandler func(context.Context, events.APIGatewayV2HTTPRequest, error) (events.APIGatewayProxyResponse, error)

// CatchAllHandler defines the function interface the router uses to handle any
// request that doesn't match a route.
type CatchAllHandler func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error)

// Router will route an incoming events.APIGatewayV2HTTPRequest the appropriate
// route based upon the router configuration and then return the
// events.APIGatewayProxyResponse.
//
// Route matching is a simple process that loops through all routes added in the
// order they were added and checks if a match is present. If so that route gets
// executed, otherwise it moves onto the next route for comparison.
//
// If the CatchAll handler is set any request that doesn't match a route will be
// handled by it.
//
// If the CatchError handler is set any route that returns an error will first
// be passed into the hander for additional processing.
//
//
// Example:
//
//	func yoloHandler(ctx *RouteContext) (events.APIGatewayProxyResponse, error) {
//		headers := map[string]string{
//			"Content-Type": "application/json",
//		}
//
//		response := events.APIGatewayProxyResponse{
//			StatusCode:      200,
//			Headers:         headers,
//			Body:            `{"yolo": "it's true"}`,
//			IsBase64Encoded: false,
//		}
//
//		return response, nil
//	}
//
//	func handler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
//		router := &proxy.Router{}
//		router.GET("/yolo", yoloHandler)
//
//		if !router.Valid() {
//			return events.APIGatewayProxyResponse{}, router.BuildErrors()
//		}
//
//		return router.Route(ctx, request)
//	}
//
type Router struct {
	Routes     []*Route
	CatchAll   CatchAllHandler
	CatchError ErrorHandler

	errors []error
}

// Valid returns true if the routers' routes have all been built successfully.
// Otherwise false.
func (router *Router) Valid() bool {
	return len(router.errors) == 0
}

// AddRoute appends route to the list of routes used for request matching.
func (router *Router) AddRoute(route *Route) {
	router.Routes = append(router.Routes, route)
}

// AddBuildError appends an error to the list of router errors.
func (router *Router) AddBuildError(err error) {
	router.errors = append(router.errors, err)
}

// BuildErrors returns a single error that encapsulates all the route errors
// found during router construction.
func (router *Router) BuildErrors() error {
	topError := errors.New("failed building router")

	for _, err := range router.errors {
		topError = errors.Wrap(topError, err.Error())
	}

	return topError
}

// AddRouteIfNoError appends the provided route if no error is present.
// Otherwise it adds the error to the build errors.
//
// This method is provided to simplify router construction with many routes by
// reducing error checking boilerplate.
func (router *Router) AddRouteIfNoError(route *Route, err error) {
	if err != nil {
		router.AddBuildError(err)
	} else {
		router.AddRoute(route)
	}
}

// GET adds a new GET route with the specified pattern match and handler.
func (router *Router) GET(match string, handler RouteHandler) {
	router.AddRouteIfNoError(NewRoute(GET, match, handler))
}

// HEAD adds a new HEAD route with the specified pattern match and handler.
func (router *Router) HEAD(match string, handler RouteHandler) {
	router.AddRouteIfNoError(NewRoute(HEAD, match, handler))
}

// POST adds a new POST route with the specified pattern match and handler.
func (router *Router) POST(match string, handler RouteHandler) {
	router.AddRouteIfNoError(NewRoute(POST, match, handler))
}

// PUT adds a new PUT route with the specified pattern match and handler.
func (router *Router) PUT(match string, handler RouteHandler) {
	router.AddRouteIfNoError(NewRoute(PUT, match, handler))
}

// DELETE adds a new DELETE route with the specified pattern match and handler.
func (router *Router) DELETE(match string, handler RouteHandler) {
	router.AddRouteIfNoError(NewRoute(DELETE, match, handler))
}

// CONNECT adds a new CONNECT route with the specified pattern match and handler.
func (router *Router) CONNECT(match string, handler RouteHandler) {
	router.AddRouteIfNoError(NewRoute(CONNECT, match, handler))
}

// OPTIONS adds a new OPTIONS route with the specified pattern match and handler.
func (router *Router) OPTIONS(match string, handler RouteHandler) {
	router.AddRouteIfNoError(NewRoute(OPTIONS, match, handler))
}

// TRACE adds a new TRACE route with the specified pattern match and handler.
func (router *Router) TRACE(match string, handler RouteHandler) {
	router.AddRouteIfNoError(NewRoute(TRACE, match, handler))
}

// PATCH adds a new PATCH route with the specified pattern match and handler.
func (router *Router) PATCH(match string, handler RouteHandler) {
	router.AddRouteIfNoError(NewRoute(PATCH, match, handler))
}

// AddCatchAllHandler attaches a catchall handler to the router.
func (router *Router) AddCatchAllHandler(handler CatchAllHandler) {
	router.CatchAll = handler
}

// AddErrorHandler attaches a error handler to the router.
func (router *Router) AddErrorHandler(handler ErrorHandler) {
	router.CatchError = handler
}

// routeInternal loops through all routes and checks if the request matches any
// of them.
//
// If there is a match it executes the route's handler.
//
// If the catch all handler is set and no route is matched it gets executed.
//
// If there is no catch all handler and no route is matched an error is returned.
func (router *Router) routeInternal(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
	for _, route := range router.Routes {
		matched, groups := route.IsMatch(request)

		if !matched {
			continue
		}

		return route.Follow(ctx, request, groups)
	}

	if router.CatchAll != nil {
		return router.CatchAll(ctx, request)
	}

	return events.APIGatewayProxyResponse{}, fmt.Errorf("'%s %s' not found", request.RequestContext.HTTP.Method, request.RawPath)
}

// Route loops through all routes and checks if the request matches any of them.
//
// If there is a match it executes the route's handler.
//
// If the catch all handler is set and no route is matched it gets executed.
//
// If there is no catch all handler and no route is matched an error is returned.
//
// If there is an error handler set and an error occurs the errors the error
// handler is executed and it's result returned.
func (router *Router) Route(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
	if router.CatchError == nil {
		return router.routeInternal(ctx, request)
	}

	response, err := router.routeInternal(ctx, request)

	if err != nil {
		return router.CatchError(ctx, request, err)
	}

	return response, nil
}
