package lambdautils

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/stretchr/testify/assert"
)

func prepareContext(fn, v, alias string) context.Context {
	lambdacontext.FunctionName = fn
	lambdacontext.FunctionVersion = v
	lambdacontext.LogGroupName = "logGroupName-test"
	lambdacontext.LogStreamName = "logStreamName-test"
	lambdacontext.MemoryLimitInMB = 100

	arn := []string{"arn:aws:lambda:us-east-1:xxxxx:function", fn}
	if alias != "" {
		arn = append(arn, alias)
	}

	lctx := lambdacontext.LambdaContext{InvokedFunctionArn: strings.Join(arn, ":")}
	return lambdacontext.NewContext(context.Background(), &lctx)
}

func clearContext() {
	lambdacontext.FunctionName = os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	lambdacontext.FunctionVersion = os.Getenv("AWS_LAMBDA_FUNCTION_VERSION")
	lambdacontext.LogGroupName = os.Getenv("AWS_LAMBDA_LOG_GROUP_NAME")
	lambdacontext.LogStreamName = os.Getenv("AWS_LAMBDA_LOG_STREAM_NAME")
	if limit, err := strconv.Atoi(os.Getenv("AWS_LAMBDA_FUNCTION_MEMORY_SIZE")); err != nil {
		lambdacontext.MemoryLimitInMB = 0
	} else {
		lambdacontext.MemoryLimitInMB = limit
	}
}

func TestLambdaMetaData(t *testing.T) {
	// NOTE: must set and unset the lambdacontext global vars. This is an
	// anti-pattern: https://dave.cheney.net/2017/06/11/go-without-package-scoped-variables
	defer clearContext()

	cases := []struct {
		fn          string
		v           string
		alias       string
		expectedArn string
	}{
		{"fname", "1", "PRODUCTION", "arn:aws:lambda:us-east-1:xxxxx:function:fname:PRODUCTION"},
		{"fname", "$LATEST", "$LATEST", "arn:aws:lambda:us-east-1:xxxxx:function:fname:$LATEST"},
		{"fname", "2", "DEV", "arn:aws:lambda:us-east-1:xxxxx:function:fname:DEV"},
		{"fname", "4", "", "arn:aws:lambda:us-east-1:xxxxx:function:fname"},
		{"fname2", "3", "PRODUCTION", "arn:aws:lambda:us-east-1:xxxxx:function:fname2:PRODUCTION"},
	}

	for _, c := range cases {
		ctx := prepareContext(c.fn, c.v, c.alias)
		meta := GetLambdaMetaData(ctx)

		assert.Equal(t, c.fn, meta.FunctionName)
		assert.Equal(t, c.v, meta.FunctionVersion)
		assert.Equal(t, 100, meta.MemoryLimitInMB)
		assert.Equal(t, "logGroupName-test", meta.LogGroupName)
		assert.Equal(t, "logStreamName-test", meta.LogStreamName)
		assert.Equal(t, c.expectedArn, meta.Context.InvokedFunctionArn)
	}
}
