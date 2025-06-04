package errxlambda

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/aws/aws-lambda-go/events"
)

// LambdaHandlerFunc represents a Lambda handler function that can return an Error
type LambdaHandlerFunc func(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

// ErrorMiddleware wraps a Lambda handler to automatically handle Error responses
func ErrorMiddleware(handler LambdaHandlerFunc) LambdaHandlerFunc {
	return func(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		response, err := handler(ctx, event)

		if err != nil {
			log.Printf("Lambda error handling request: %v", err)

			var xerr *errx.Error
			if errors.As(err, &xerr) {
				return ToLambdaResponse(xerr), nil
			}

			unknownErr := &errx.Error{
				Code:       "INTERNAL_ERROR",
				Type:       errx.TypeInternal,
				Message:    err.Error(),
				HTTPStatus: 500,
			}
			return ToLambdaResponse(unknownErr), nil
		}

		return response, nil
	}
}

// ToLambdaResponse converts an errx.Error to an API Gateway proxy response
func ToLambdaResponse(e *errx.Error) events.APIGatewayProxyResponse {
	errorResponse := map[string]any{
		"code":    e.Code,
		"type":    e.Type,
		"message": e.Message,
	}

	if e.Details != nil && len(e.Details) > 0 {
		errorResponse["details"] = e.Details
		log.Printf("Error details: %v", e.Details)
	}

	responseBody := map[string]any{
		"error": errorResponse,
	}

	body, err := json.Marshal(responseBody)
	if err != nil {
		log.Printf("Failed to marshal error response: %v", err)
		body = []byte(`{"error": {"code": "INTERNAL_ERROR", "type": "INTERNAL", "message": "An unexpected error occurred"}}`)
	}

	status := e.HTTPStatus
	if status == 0 {
		status = 500
	}

	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(body),
	}
}

// JSONHandlerFunc represents a Lambda handler that works with JSON request/response
type JSONHandlerFunc[TRequest, TResponse any] func(ctx context.Context, req TRequest) (TResponse, error)

// JSONErrorMiddleware wraps a JSON-based Lambda handler
func JSONErrorMiddleware[TRequest, TResponse any](handler JSONHandlerFunc[TRequest, TResponse]) LambdaHandlerFunc {
	return func(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		var req TRequest
		if event.Body != "" {
			if err := json.Unmarshal([]byte(event.Body), &req); err != nil {
				parseErr := &errx.Error{
					Code:       "INVALID_REQUEST_BODY",
					Type:       errx.TypeValidation,
					Message:    "Failed to parse request body",
					HTTPStatus: 400,
					Cause:      err,
				}
				return ToLambdaResponse(parseErr), nil
			}
		}

		response, err := handler(ctx, req)
		if err != nil {
			log.Printf("Lambda JSON handler error: %v", err)

			var xerr *errx.Error
			if errors.As(err, &xerr) {
				return ToLambdaResponse(xerr), nil
			}

			unknownErr := &errx.Error{
				Code:       "INTERNAL_ERROR",
				Type:       errx.TypeInternal,
				Message:    err.Error(),
				HTTPStatus: 500,
			}
			return ToLambdaResponse(unknownErr), nil
		}

		body, err := json.Marshal(response)
		if err != nil {
			marshalErr := &errx.Error{
				Code:       "RESPONSE_MARSHAL_ERROR",
				Type:       errx.TypeInternal,
				Message:    "Failed to marshal response",
				HTTPStatus: 500,
				Cause:      err,
			}
			return ToLambdaResponse(marshalErr), nil
		}

		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: string(body),
		}, nil
	}
}
