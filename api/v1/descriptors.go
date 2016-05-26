package v1

import (
	"net/http"
	"regexp"

	"github.com/danielkrainas/canaria-api/api/describe"
	"github.com/danielkrainas/canaria-api/api/errcode"
)

var (
	IdRegex = regexp.MustCompile(`(?i)[0-9a-f]{12}4[0-9a-f]{3}[89ab][0-9a-f]{15}\\Z`)

	uuidParameter = describe.ParameterDescriptor{
		Name:        "uuid",
		Type:        "string",
		Required:    true,
		Description: "A uuid identifying the canary",
	}

	hostHeader = describe.ParameterDescriptor{
		Name:        "Host",
		Type:        "string",
		Description: "",
		Format:      "<hostname>",
		Examples:    []string{"api.canaria.io"},
	}

	authHeader = describe.ParameterDescriptor{
		Name:        "Authorization",
		Type:        "string",
		Description: "An RFC7235 compliant authorization header.",
		Format:      "<scheme> <token>",
		Examples:    []string{"Bearer eHZ6MWV2RlM0d0VFUFRHRUZQSEJvZ=="},
	}

	jsonContentLengthHeader = describe.ParameterDescriptor{
		Name:        "Content-Length",
		Type:        "integer",
		Description: "Length of the JSON body.",
		Format:      "<length>",
	}

	zeroContentLengthHeader = describe.ParameterDescriptor{
		Name:        "Content-Length",
		Type:        "integer",
		Description: "The 'Content-Length' header must be zero and the body must be empty.",
		Format:      "0",
	}

	authChallengeHeader = describe.ParameterDescriptor{
		Name:        "WWW-Authenticate",
		Type:        "string",
		Description: "An RFC 7235 compliant authentication challenge header.",
		Format:      `<scheme> realm="<realm>", ...`,
		Examples: []string{
			`Bearer realm="https://auth.canaria.io/", service="api.canaria.io", scopes="canary:1f07fe68-4161-4617-808b-52a0bcf41b39:kill"`,
		},
	}

	unauthorizedResponseDescriptor = describe.ResponseDescriptor{
		Name:        "Authentication Required",
		StatusCode:  http.StatusUnauthorized,
		Description: "The client is not authenticated.",
		Headers: []describe.ParameterDescriptor{
			authChallengeHeader,
			jsonContentLengthHeader,
		},

		Body: describe.BodyDescriptor{
			ContentType: "application/json; charset=utf-8",
			Format:      errorsBody,
		},
		ErrorCodes: []errcode.ErrorCode{
			errcode.ErrorCodeUnauthorized,
		},
	}

	canaryNotFoundResponseDescriptor = describe.ResponseDescriptor{
		Name:        "No Such Canary Error",
		StatusCode:  http.StatusNotFound,
		Description: "The canary is not known to the server",
		Headers: []describe.ParameterDescriptor{
			jsonContentLengthHeader,
		},

		Body: describe.BodyDescriptor{
			ContentType: "application/json; charset=utf-8",
			Format:      errorsBody,
		},

		ErrorCodes: []errcode.ErrorCode{
			ErrorCodeCanaryUnknown,
		},
	}

	deadResponseDescriptor = describe.ResponseDescriptor{
		Name:        "Dead Canary Error",
		StatusCode:  http.StatusNotFound,
		Description: "The canary existed but is now dead.",
		Headers: []describe.ParameterDescriptor{
			jsonContentLengthHeader,
		},

		Body: describe.BodyDescriptor{
			ContentType: "application/json; charset=utf-8",
			Format:      errorsBody,
		},

		ErrorCodes: []errcode.ErrorCode{
			ErrorCodeCanaryDead,
		},
	}
)

var (
	errorsBody = ``

	canaryRequestBody = ``

	canaryBody = ``
)

var APIDescriptor = struct {
	RouteDescriptors []describe.RouteDescriptor
}{
	RouteDescriptors: routeDescriptors,
}

var routeDescriptors = []describe.RouteDescriptor{
	{
		Name:        RouteNameBase,
		Path:        "/v1",
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
								Headers: []describe.ParameterDescriptor{
									zeroContentLengthHeader,
								},
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
		Name:        RouteNameCanaries,
		Path:        "/v1/canaries",
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

						Body: describe.BodyDescriptor{
							ContentType: "",
							Format:      canaryRequestBody,
						},

						Successes: []describe.ResponseDescriptor{
							{
								Description: "The canary was accepted by the server.",
								StatusCode:  http.StatusCreated,
								Headers: []describe.ParameterDescriptor{
									{
										Name:        "Location",
										Description: "The canonical location url of the created canary.",
										Type:        "url",
										Format:      "<url>",
										Examples:    []string{"https://api.canaria.io/canary/07034f6b-8604-470c-8609-21a79ed0c56b"},
									},
									zeroContentLengthHeader,
								},
							},
						},

						Failures: []describe.ResponseDescriptor{
							{
								Name:        "Not allowed",
								Description: "Canary put is not allowed because the server is configured for read-only.",
								StatusCode:  http.StatusMethodNotAllowed,
								ErrorCodes: []errcode.ErrorCode{
									errcode.ErrorCodeUnsupported,
								},
							},
							{
								Name:        "Invalid Canary",
								Description: "The received canary settings were invalid in some way as described by the error codes. The client should resolve the issue and retry the request.",
								StatusCode:  http.StatusBadRequest,
								Body: describe.BodyDescriptor{
									ContentType: "application/json; charset=utf-8",
									Format:      errorsBody,
								},
								ErrorCodes: []errcode.ErrorCode{
									ErrorCodeTTLInvalid,
								},
							},
							unauthorizedResponseDescriptor,
						},
					},
				},
			},
		},
	},
	{
		Name:        RouteNameCanary,
		Path:        "/v1/canary/{canary_id:" + IdRegex.String() + "}",
		Entity:      "Canary",
		Description: "",
		Methods: []describe.MethodDescriptor{
			{
				Method:      "GET",
				Description: "",
				Requests: []describe.RequestDescriptor{
					{
						Name:        "Canary",
						Description: "Return a canary",
						Headers: []describe.ParameterDescriptor{
							hostHeader,
							authHeader,
						},

						PathParameters: []describe.ParameterDescriptor{
							uuidParameter,
						},

						Successes: []describe.ResponseDescriptor{
							{
								Description: "",
								StatusCode:  http.StatusOK,
								Headers: []describe.ParameterDescriptor{
									jsonContentLengthHeader,
								},

								Body: describe.BodyDescriptor{
									ContentType: "application/json; charset=utf-8",
									Format:      canaryBody,
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
				Method:      "HEAD",
				Description: "",
				Requests: []describe.RequestDescriptor{
					{
						Name:        "CanaryCheck",
						Description: "",
						Headers: []describe.ParameterDescriptor{
							hostHeader,
							authHeader,
						},

						PathParameters: []describe.ParameterDescriptor{
							uuidParameter,
						},

						Successes: []describe.ResponseDescriptor{
							{
								Description: "Healthy Canary",
								StatusCode:  http.StatusNoContent,
								Headers: []describe.ParameterDescriptor{
									zeroContentLengthHeader,
								},
							},
						},

						Failures: []describe.ResponseDescriptor{
							{
								Description: "Invalid Canary",
								StatusCode:  http.StatusNotFound,
								Headers: []describe.ParameterDescriptor{
									zeroContentLengthHeader,
								},
								ErrorCodes: []errcode.ErrorCode{
									ErrorCodeCanaryDead,
									ErrorCodeCanaryUnknown,
								},
							},
							{
								Description: "Access Denied",
								StatusCode:  http.StatusForbidden,
								Headers: []describe.ParameterDescriptor{
									zeroContentLengthHeader,
								},
								ErrorCodes: []errcode.ErrorCode{
									errcode.ErrorCodeDenied,
								},
							},
						},
					},
				},
			},
			{
				Method:      "DELETE",
				Description: "",
				Requests: []describe.RequestDescriptor{
					{
						Headers: []describe.ParameterDescriptor{
							hostHeader,
							authHeader,
						},

						PathParameters: []describe.ParameterDescriptor{
							uuidParameter,
						},

						Successes: []describe.ResponseDescriptor{
							{
								StatusCode: http.StatusAccepted,
								Headers: []describe.ParameterDescriptor{
									zeroContentLengthHeader,
								},
							},
						},

						Failures: []describe.ResponseDescriptor{
							deadResponseDescriptor,
							canaryNotFoundResponseDescriptor,
							unauthorizedResponseDescriptor,
						},
					},
				},
			},
		},
	},
	{
		Name:        RouteNameWebhooks,
		Path:        "/v1/canary/{canary_id:" + IdRegex.String() + "}/hooks",
		Entity:      "Webhook",
		Description: "",
		Methods: []describe.MethodDescriptor{
			{
				Method:      "PUT",
				Description: "",
				Requests: []describe.RequestDescriptor{
					{
						Headers: []describe.ParameterDescriptor{},
					},
				},
			},
		},
	},
	{
		Name:        RouteNameWebhook,
		Path:        "/v1/canary/{canary_id:" + IdRegex.String() + "}/hooks/{hook_id:" + IdRegex.String() + "}",
		Entity:      "Webhook",
		Description: "",
		Methods: []describe.MethodDescriptor{
			{
				Method:      "DELETE",
				Description: "",
				Requests: []describe.RequestDescriptor{
					{},
				},
			},
			{
				Method:      "POST",
				Description: "",
				Requests: []describe.RequestDescriptor{
					{},
				},
			},
		},
	},
	{
		Name:        RouteNameWebhookTest,
		Path:        "/v1/canary/{canary_id:" + IdRegex.String() + "}/hooks/{hook_id:" + IdRegex.String() + "}/ping",
		Entity:      "Webhook",
		Description: "",
		Methods: []describe.MethodDescriptor{
			{
				Method:      "GET",
				Description: "",
				Requests: []describe.RequestDescriptor{
					{},
				},
			},
		},
	},
}
