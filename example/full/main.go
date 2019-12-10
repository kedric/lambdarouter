package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/kedric/lambdarouter"
)

var variables = lambdarouter.StageVariables{
	"stagename": {
		"variablename": "value",
	},
}

func authorizer(ctx context.Context, request events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	println("ok")
	arnSplit := strings.Split(request.MethodArn, ":")
	apiGatewayArnTmp := strings.Split(arnSplit[5], "/")
	accessArn := fmt.Sprintf(`arn:aws:execute-api:%s:%s:%s/*/*/*`, arnSplit[3], arnSplit[4], apiGatewayArnTmp[0])
	return events.APIGatewayCustomAuthorizerResponse{
		PrincipalID: strconv.Itoa(1),
		Context:     map[string]interface{}{},
		PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				events.IAMPolicyStatement{
					Action:   []string{"execute-api:Invoke"},
					Effect:   "Allow",
					Resource: []string{accessArn},
				},
			},
		},
	}, nil
}

func Index(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Welcome!\n",
	}, nil
}

func Hello(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       fmt.Sprintf("hello, %s!\n", req.PathParameters["name"]),
	}, nil
}

func main() {
	router := lambdarouter.New()
	router.SetAuthorizer(authorizer)
	router.GET("/", Index)
	router.GET("/hello/:name", Hello)
	router.Serve(":8081", variables)
}
