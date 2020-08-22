// Package form implements reflection-based HTTP request decoder.
//
// Request structure is the single source of truth about handler expectations.
//
// It is a `Go` type reflected to produce `Swagger` and `JSON` schemas.
//
// When `*http.Request` comes, the values that it holds are assigned into a new instance of handler request structure.
// Mapping of data is based of field tag values:
//
//  - `query` for URL query data,
//  - `path` for parameters from URL path defined by handler pattern (e.g. `/categories/{id}`),
//  - `header` for parameters in request header,
//  - `cookie` for parameters in request cookie,
//  - `file` for file uploads,
//  - `json` for JSON request body.
//
// For additional `JSON Schema` validation you can provide more field tags: https://godoc.org/github.com/swaggest/swgen#hdr-JSON_Schema_tags.
//
// Sample request structure:
//
//   // DefaultCompensationGetRequest defines query parameters options search.
//   type DefaultCompensationGetRequest struct {
//       CategoryID uuid.UUID `path:"id"`
//
//       Country hf.Country `query:"country"`
//
//       OrderID       string `query:"order_id" required:"false"`
//       IngredientSKU string `query:"ingredient_sku" required:"false"`
//   }
//
//
// Mapping process includes `JSON Schema` based validation.
//
// If validation fails `Bad Request` status is being responded and actual handler is not called.
//
// Prepared request data is stored in request context, later it can be retrieved in handler.
//
//   input, ok := request.InputFromContext(req.Context()).(*DefaultCompensationGetRequest)
//   if !ok {
//   	panic("uh-oh")
//   }
//   query := input.(*DefaultCompensationGetRequest)
//
package decoder
