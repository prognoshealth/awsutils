package lambdautils

import (
	"context"

	"github.com/aws/aws-lambda-go/lambdacontext"
)

// LambdaMetaData stored details about the current lambda context.
type LambdaMetaData struct {
	FunctionName    string
	FunctionVersion string
	LogGroupName    string
	LogStreamName   string
	MemoryLimitInMB int
	Context         *lambdacontext.LambdaContext
}

// GetLambdaMetaData returns MetaData extracted from the current lambda context.
func GetLambdaMetaData(ctx context.Context) LambdaMetaData {
	lm := LambdaMetaData{
		FunctionName:    lambdacontext.FunctionName,
		FunctionVersion: lambdacontext.FunctionVersion,
		LogGroupName:    lambdacontext.LogGroupName,
		LogStreamName:   lambdacontext.LogStreamName,
		MemoryLimitInMB: lambdacontext.MemoryLimitInMB,
	}

	lm.Context, _ = lambdacontext.FromContext(ctx)
	return lm
}
