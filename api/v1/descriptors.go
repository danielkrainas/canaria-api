package v1

import (
	"net/http"
	"regexp"

	"github.com/danielkrainas/canaria-api/api/describe"
	"github.com/danielkrainas/canaria-api/api/errorcode"
)

var (
	uuidParameter = describe.ParameterDescriptor{
		Name:        "uuid",
		Type:        "string",
		Required:    true,
		Description: "A uuid identifying the canary",
	}
)

var APIDescriptor = struct {
	RouteDescriptors []describe.RouteDescriptor
}{
	RouteDescriptors: routeDescriptors,
}

var routeDescriptors = []describe.RouteDescriptor{
	{
		Name:        "base",
		Path:        "/v1/",
		Entity:      "Base",
		Description: "Base V1 API route, can be used for lightweight version checks and to validate authentication.",
		Methods: []describe.MethodDescriptor{
			{
				Method:      "GET",
				Description: "Check that the server supports the Canaria V1 API.",
				Requests: []describe.RequestDescriptor{
					{
						Headers: []describe.ParameterDescriptor{
							hostHeader,
							authHeader,
						},

						Successes: []describe.ResponseDescriptor{
							{
								Description: "The API implements the V1 protocol and is accessible.",
								StatusCode:  http.StatusOK,
							},
						},

						Failures: []describe.ResponseDescriptor{
							{
								Description: "The API does not support the V1 protocol.",
								StatusCode:  http.StatusNotFound,
							},
							unauthorizedResponseDescriptor,
						},
					},
				},
			},
		},
	},
}
