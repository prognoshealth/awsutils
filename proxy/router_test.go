package proxy

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestRouter_Valid_true(t *testing.T) {
	r := &Router{}

	assert.True(t, r.Valid())
}

func TestRouter_Valid_false(t *testing.T) {
	r := &Router{}
	r.AddBuildError(errors.New("some error"))

	assert.False(t, r.Valid())
}

func TestRouter_AddRoute(t *testing.T) {
	r := &Router{}

	assert.Empty(t, r.Routes)

	route, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	r.AddRoute(route)

	assert.Len(t, r.Routes, 1)
	assert.Equal(t, route, r.Routes[0])

	route2, err := NewRoute(GET, "/yolo2", testHandler)
	assert.NoError(t, err)

	r.AddRoute(route2)

	assert.Len(t, r.Routes, 2)
	assert.Equal(t, route2, r.Routes[1])
}

func TestRouter_AddBuildError(t *testing.T) {
	r := &Router{}

	assert.Empty(t, r.errors)

	r.AddBuildError(errors.New("some error"))

	assert.Len(t, r.errors, 1)
	assert.Equal(t, "some error", r.errors[0].Error())

	r.AddBuildError(errors.New("some other error"))

	assert.Len(t, r.errors, 2)
	assert.Equal(t, "some other error", r.errors[1].Error())
}

func TestRouter_BuildErrors(t *testing.T) {
	r := &Router{}

	r.AddBuildError(errors.New("some error"))
	r.AddBuildError(errors.New("some other error"))

	err := r.BuildErrors()

	assert.Equal(t, "some other error: some error: failed building router", err.Error())
}

func TestRouter_AddRouteIfNoError(t *testing.T) {
	r := &Router{}

	r.AddRouteIfNoError(NewRoute(GET, "/yolo", testHandler))
	r.AddRouteIfNoError(NewRoute(GET, "asom (?<in-invalid>.*)", testHandler))

	assert.Len(t, r.Routes, 1)
	assert.Len(t, r.errors, 1)

	assert.Equal(t, "GET ^/yolo/?$", r.Routes[0].String())

	err := r.BuildErrors()
	assert.Equal(t, "failed compiling regex pattern 'asom (?<in-invalid>.*)': error parsing regexp: invalid or unsupported Perl syntax: `(?<`: failed building router", err.Error())
}

func TestRouter_ConvenienceMethods(t *testing.T) {
	r := &Router{}
	r.GET("/route", testHandler)
	r.HEAD("/route", testHandler)
	r.POST("/route", testHandler)
	r.PUT("/route", testHandler)
	r.DELETE("/route", testHandler)
	r.CONNECT("/route", testHandler)
	r.OPTIONS("/route", testHandler)
	r.TRACE("/route", testHandler)
	r.PATCH("/route", testHandler)

	assert.Len(t, r.Routes, 9)
	assert.Equal(t, "GET ^/route/?$", r.Routes[0].String())
	assert.Equal(t, "HEAD ^/route/?$", r.Routes[1].String())
	assert.Equal(t, "POST ^/route/?$", r.Routes[2].String())
	assert.Equal(t, "PUT ^/route/?$", r.Routes[3].String())
	assert.Equal(t, "DELETE ^/route/?$", r.Routes[4].String())
	assert.Equal(t, "CONNECT ^/route/?$", r.Routes[5].String())
	assert.Equal(t, "OPTIONS ^/route/?$", r.Routes[6].String())
	assert.Equal(t, "TRACE ^/route/?$", r.Routes[7].String())
	assert.Equal(t, "PATCH ^/route/?$", r.Routes[8].String())
}

func TestRouter_AddCatchAllHandler(t *testing.T) {
	r := &Router{}

	assert.Nil(t, r.CatchAll)

	handler := func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{
			StatusCode:      404,
			Headers:         map[string]string{},
			Body:            "some body",
			IsBase64Encoded: false,
		}, nil
	}

	r.AddCatchAllHandler(handler)

	response, err := r.CatchAll(context.Background(), events.APIGatewayV2HTTPRequest{})
	assert.NoError(t, err)
	assert.Equal(t, 404, response.StatusCode)
}

func TestRouter_AddErrorHandler(t *testing.T) {
	r := &Router{}

	assert.Nil(t, r.CatchError)

	handler := func(ctx context.Context, request events.APIGatewayV2HTTPRequest, err error) (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{
			StatusCode:      404,
			Headers:         map[string]string{},
			Body:            err.Error(),
			IsBase64Encoded: false,
		}, nil
	}

	r.AddErrorHandler(handler)

	response, err := r.CatchError(context.Background(), events.APIGatewayV2HTTPRequest{}, errors.New("some error"))
	assert.NoError(t, err)
	assert.Equal(t, "some error", response.Body)
}

func TestRouter_Route(t *testing.T) {
	r := &Router{}

	routeHandler := func(context *RouteContext) (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	r.GET("/route", routeHandler)

	request := testRequest(GET, "/route")
	response, err := r.Route(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
}

func TestRouter_Route_multiple(t *testing.T) {
	r := &Router{}

	routeHandler := func(context *RouteContext) (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{
				StatusCode: 200,
				Body:       context.Request.RawPath,
			},
			nil
	}

	r.GET("/route", routeHandler)
	r.GET("/route2", routeHandler)
	r.GET("/route/to/(?P<id>[0-9]+)/something", routeHandler)

	request := testRequest(GET, "/route2")
	response, err := r.Route(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "/route2", response.Body)

	request = testRequest(GET, "/route/to/5/something")
	response, err = r.Route(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "/route/to/5/something", response.Body)
}

func TestRouter_Route_params(t *testing.T) {
	r := &Router{}

	routeHandler := func(context *RouteContext) (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{
				StatusCode: 200,
				Body:       context.Params["id"],
			},
			nil
	}

	r.GET("/route/to/(?P<id>[0-9]+)/something", routeHandler)

	request := testRequest(GET, "/route/to/5/something")
	response, err := r.Route(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "5", response.Body)

	request = testRequest(GET, "/route/to/42/something")
	response, err = r.Route(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "42", response.Body)
}

func TestRouter_Route_catchError_error(t *testing.T) {
	r := &Router{}

	errorHandler := func(ctx context.Context, request events.APIGatewayV2HTTPRequest, err error) (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{
			StatusCode:      500,
			Headers:         map[string]string{},
			Body:            err.Error(),
			IsBase64Encoded: false,
		}, nil
	}

	routeHandler := func(context *RouteContext) (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{StatusCode: 500}, errors.New("failed")
	}

	r.GET("/route", routeHandler)
	r.AddErrorHandler(errorHandler)

	request := testRequest(GET, "/route")

	response, err := r.Route(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, 500, response.StatusCode)
	assert.Equal(t, "failed", response.Body)
}

func TestRouter_Route_catchError_noError(t *testing.T) {
	r := &Router{}

	errorHandler := func(ctx context.Context, request events.APIGatewayV2HTTPRequest, err error) (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{
			StatusCode:      500,
			Headers:         map[string]string{},
			Body:            err.Error(),
			IsBase64Encoded: false,
		}, nil
	}

	routeHandler := func(context *RouteContext) (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	r.GET("/route", routeHandler)
	r.AddErrorHandler(errorHandler)

	request := testRequest(GET, "/route")

	response, err := r.Route(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
}

func TestRouter_Route_noCatchError_error(t *testing.T) {
	r := &Router{}

	routeHandler := func(context *RouteContext) (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{StatusCode: 500}, errors.New("failed")
	}

	r.GET("/route", routeHandler)

	request := testRequest(GET, "/route")

	response, err := r.Route(context.Background(), request)

	assert.Error(t, err)
	assert.Equal(t, "failed", err.Error())
	assert.Equal(t, 500, response.StatusCode)
}

func TestRouter_Route_noCatchError_noError(t *testing.T) {
	r := &Router{}

	routeHandler := func(context *RouteContext) (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	r.GET("/route", routeHandler)

	request := testRequest(GET, "/route")

	response, err := r.Route(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
}

func TestRouter_Route_noCatchAll_noMatch(t *testing.T) {
	r := &Router{}

	request := testRequest(GET, "/yolo")
	_, err := r.Route(context.Background(), request)

	assert.Error(t, err)
	assert.Equal(t, "'GET /yolo' not found", err.Error())
}

func TestRouter_Route_CatchAll_noMatch(t *testing.T) {
	r := &Router{}

	handler := func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{
			StatusCode:      404,
			Headers:         map[string]string{},
			Body:            "not found",
			IsBase64Encoded: false,
		}, nil
	}

	r.AddCatchAllHandler(handler)

	request := testRequest(GET, "/yolo")
	response, err := r.Route(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, 404, response.StatusCode)
	assert.Equal(t, "not found", response.Body)
}
