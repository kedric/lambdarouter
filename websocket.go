package lambdarouter

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

type WebsocketHandler func(context context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error)

type WebsocketMux struct {
	wsevent                     map[string]WebsocketHandler
	templateSelectionExpression string
}

func (ws *WebsocketMux) On(eventName string, handler WebsocketHandler) {
	ws.wsevent[eventName] = handler
}

func ReformateTemplateSelectionExpression(original string) string {
	tmp := strings.ReplaceAll(original, "$request.body", "{{.body")
	tmp = strings.ReplaceAll(tmp, "${request.body", "{{.body")
	tmp = strings.ReplaceAll(tmp, "}", "}}")

	if strings.Contains(tmp, "{{") && !strings.Contains(tmp, "}}") {
		tmp += "}}"
	}
	return tmp
}

func ResolveTemplateSelectionExpression(original string, request map[string]interface{}) string {
	_request := request
	if vs, ok := _request["body"].(string); ok && _request["body"] != nil {
		tmp := map[string]interface{}{}
		if err := json.Unmarshal([]byte(vs), &tmp); err == nil {
			_request["body"] = tmp
		}
	}
	tmpl, err := template.New("test").Parse(ReformateTemplateSelectionExpression(original))
	if err != nil {
		panic(err)
	}
	b := bytes.NewBuffer([]byte{})
	tmpl.Execute(b, _request)
	// tmpl.Execute(b, map[string]map[string]interface{}{"body": body})
	return string(b.Bytes())
}

func (ws WebsocketMux) dispatch(ctx context.Context, ev map[string]interface{}) (events.APIGatewayProxyResponse, error) {
	event := toWsEvent(ev)
	// eventName := ResolveTemplateSelectionExpression(ws.templateSelectionExpression, ev)
	switch event.RequestContext.RouteKey {
	case "$connect":
	case "$disconnect":
	case "$default":
	default:

	}
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "OK",
	}, nil
}

func NewWebsocket() *WebsocketMux {
	return &WebsocketMux{wsevent: map[string]WebsocketHandler{}}
}

func isWebsocketEvent(ev map[string]interface{}) bool {
	// println("tesst resolve", ResolveTemplateSelectionExpression("$request.body.action", ev))
	if v, ok := ev["requestContext"]; ok {
		if vi, ok := v.(map[string]interface{}); ok {
			if _, ok := vi["eventType"]; ok {
				return true
			}
		}
	}
	return false
}
