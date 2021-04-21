package proxy

import (
	"context"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func testHandler(context *RouteContext) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func testRequest(method HttpMethod, path string) events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{
		RawPath: path,
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: method.String(),
			},
		},
	}
}

func TestNewRoute(t *testing.T) {
	r, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	assert.Equal(t, GET, r.Method)
	assert.True(t, r.Regex.MatchString("/yolo"))
	assert.False(t, r.Regex.MatchString("/yolo/somethingelse"))
	assert.NotNil(t, r.Handler)
}

func TestNewRoute_Error(t *testing.T) {
	_, err := NewRoute(GET, "asom (?<in-invalid>.*)", testHandler)
	assert.Error(t, err)
}

func TestRoute_Match(t *testing.T) {
	r, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	request := testRequest(GET, "/yolo")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)
	assert.Equal(t, []string{"/yolo"}, groups)
}

func TestRoute_wild(t *testing.T) {
	r, err := NewRoute(OPTIONS, ".*", testHandler)
	assert.NoError(t, err)

	request := testRequest(OPTIONS, "/yolo")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)
	assert.Equal(t, []string{"/yolo"}, groups)
}

func TestRoute_Match_trailingSlash(t *testing.T) {
	r, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	request := testRequest(GET, "/yolo/")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)
	assert.Equal(t, []string{"/yolo/"}, groups)
}

func TestRoute_Match_groups(t *testing.T) {
	r, err := NewRoute(GET, "/yolo/(?P<key>[^/]+)", testHandler)
	assert.NoError(t, err)

	request := testRequest(GET, "/yolo/the-id")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)
	assert.Equal(t, []string{"/yolo/the-id", "the-id"}, groups)
}

func TestRoute_Match_nope(t *testing.T) {
	r, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	request := testRequest(GET, "/something-else")
	matched, groups := r.IsMatch(request)

	assert.False(t, matched)
	assert.Nil(t, groups)
}

func TestRoute_Match_nopeMethod(t *testing.T) {
	r, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	request := testRequest(POST, "/yolo")
	matched, groups := r.IsMatch(request)

	assert.False(t, matched)
	assert.Nil(t, groups)
}

func TestRoute_Context(t *testing.T) {
	r, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	ctx := context.Background()
	request := testRequest(GET, "/yolo")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)

	rctx, error := r.Context(ctx, request, groups)

	assert.NoError(t, error)
	assert.Equal(t, ctx, rctx.Context)
	assert.Equal(t, request, rctx.Request)
	assert.Empty(t, rctx.Params)
}

func TestRoute_Context_wild(t *testing.T) {
	r, err := NewRoute(OPTIONS, ".*", testHandler)
	assert.NoError(t, err)

	ctx := context.Background()
	request := testRequest(OPTIONS, "/yolo")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)

	rctx, error := r.Context(ctx, request, groups)

	assert.NoError(t, error)
	assert.Equal(t, ctx, rctx.Context)
	assert.Equal(t, request, rctx.Request)
	assert.Empty(t, rctx.Params)
}

func TestRoute_Context_params(t *testing.T) {
	r, err := NewRoute(GET, "/yolo/(?P<id>[0-9]+)", testHandler)
	assert.NoError(t, err)

	ctx := context.Background()
	request := testRequest(GET, "/yolo/4")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)

	rctx, error := r.Context(ctx, request, groups)

	expected := map[string]string{
		"id": "4",
	}

	assert.NoError(t, error)
	assert.Equal(t, ctx, rctx.Context)
	assert.Equal(t, request, rctx.Request)
	assert.Equal(t, expected, rctx.Params)
}

func TestRoute_Context_params2(t *testing.T) {
	r, err := NewRoute(GET, "/yolo/(?P<id>[0-9]+)/fetch/(?P<state>[^/]+)", testHandler)
	assert.NoError(t, err)

	ctx := context.Background()
	request := testRequest(GET, "/yolo/4/fetch/ny")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)

	rctx, error := r.Context(ctx, request, groups)

	expected := map[string]string{
		"id":    "4",
		"state": "ny",
	}

	assert.NoError(t, error)
	assert.Equal(t, ctx, rctx.Context)
	assert.Equal(t, request, rctx.Request)
	assert.Equal(t, expected, rctx.Params)
}

func TestRoute_Follow(t *testing.T) {
	r, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	ctx := context.Background()
	request := testRequest(GET, "/yolo")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)

	response, err := r.Follow(ctx, request, groups)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
}

func TestRoute_Follow_error(t *testing.T) {
	r, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	ctx := context.Background()
	request := testRequest(GET, "/yolo")

	_, err = r.Follow(ctx, request, []string{})
	assert.Error(t, err)
}
