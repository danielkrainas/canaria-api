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
	{
		Name:        "canaries",
		Path:        "/v1/canary/",
		Entity:      "Canary",
		Description: "",
		Methods: []describe.MethodDescriptor{
			{
				Method:      "PUT",
				Description: "",
				Requests: []describe.RequestDescriptor{
					{
						Headers: []describe.ParameterDescriptor{
							hostHeader,
							authHeader,
						},

						Successes: []describe.ResponseDescriptor{
							{
								Description: "The canary was accepted by the server.",
								StatusCode:  http.StatusOK,
							},
						},

						Failures: []describe.ResponseDescriptor{
							{
								Name: "Not allowed",
								Description: "Canary put is not allowed because the server is configured for read-only.",
								StatusCode: http.StatusMethodNotAllowed,
								ErrorCodes: []errcode.ErrorCode{
									errcode.ErrorCodeUnsupported,
								},
							},
							{
								Name: "Invalid Canary",
								Description: "The received canary settings were invalid in some way as described by the error codes. The client should resolve the issue and retry the request.",
								StatusCode: http.StatusBadRequest,
								Body: describe.BodyDescriptor{
									ContentType: "application/json; charset=utf-8",
									Format: errorsBody,
								},
								ErrorCodes: []errcode.ErrorCode{
									ErrorCodeTTLInvalid,
								}
							},
							unauthorizedResponseDescriptor,
						},
					},
				},
			},
		},
	},
	{
		Name: "canary",
		Path: "/v1/canary/{id:" + IdRegex.String() + "}",
		Entity: "Canary",
		Description: "",
		Methods: []describe.MethodDescriptor{
			{
				Method: "GET",
				Description: "",
				Requests: []describe.RequestDescriptor{
					{
						Name: "Canary",
						Description: "Return a canary"
						Headers: []describe.ParameterDescriptor{
							hostHeader,
						},

						Successes: []describe.ResponseDescriptor{
							{
								Description: "",
								StatusCode: http.StatusOK,
								Headers: []describe.ParameterDescriptor{
									{
										Name: "Content-Length",
										Type: "integer",
										Description: "Length of the JSON body.",
										Format: "<length>",
									},
								},

								Body: describe.BodyDescriptor{
									ContentType: "application/json; charset=utf-8",
									Format: `{

}`,
								},
							},
						},

						Failures: []describe.ResponseDescriptor{
							deadResponseDescriptor,
							canaryNotFoundResponseDescriptor,
						},
					},
				},
			},
			{
				Method: "HEAD",
				Description: "",
				Requests: []describe.RequestDescriptor{
					{
						Name: "CanaryCheck",
						Description: "",
						Headers: []describe.ParameterDescriptor{
							hostHeader,
						},

						Successes: []describe.ResponseDescriptor{
							{
								Description: "",
								StatusCode: http.StatusNoContent,
							},
							canaryNotFoundResponseDescriptor,
							deadResponseDescriptor,
						},
					},
				},
			},
		},
	},
	{
		Name: "webhooks",
		Path: "/v1/canary/{id:" + IdRegex.String() + "}/hooks",
		Entity: "WebHook",
		Description: "",
		Methods: []describe.MethodDescriptor{
			{
				Method: "POST",
				Description: "",
				Requests: []describe.RequestDescriptor{
					{

					}
				},
			},
		},
	},
	{
		Name: "webhook",
		Path: "/v1/canary/{id:" + IdRegex.String() + "}/hooks/{id:" + IdRegex.String() + "}",
		Entity: "WebHook",
		Description: "",
		Methods: []describe.MethodDescriptor{
			{
				Method: "DELETE",
				Description: "",
				Requests: []describe.RequestDescriptor{
					{

					},
				},
			},
			{
				Method: "POST",
				Description: "",
				Requests: []describe.RequestDescriptor{
					{

					},
				},
			},
		},
	},
	{
		Name: "webhook ping",
		Path: "/v1/canary/{id:" + IdRegex.String() + "}/hooks/{id:" + IdRegex.String() + "}/ping",
		Entity: "WebHook",
		Description: "",
		Methods: []describe.MethodDescriptor{
			{
				Method: "GET",
				Description: "",
				Requests: []describe.RequestDescriptor{
					{

					}
				},
			},
		},
	},
}
