// Package docs embeds the OpenAPI specification for the Retrocast API.
package docs

import _ "embed"

//go:embed openapi.yaml
var OpenAPISpec []byte
