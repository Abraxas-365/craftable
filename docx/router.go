package docx

import (
	"github.com/gofiber/fiber/v2"
)

type RouterDoc struct {
	BasePath  string      `json:"basePath"`
	Endpoints []*Endpoint `json:"endpoints"`
}

func NewRouterDoc(basePath string) *RouterDoc {
	return &RouterDoc{
		BasePath:  basePath,
		Endpoints: []*Endpoint{},
	}
}

func (r *RouterDoc) AddEndpoint(endpoint *Endpoint) *RouterDoc {
	r.Endpoints = append(r.Endpoints, endpoint)
	return r
}

// RegisterWithFiber hooks this documentation into a Fiber app
// so documentation can be accessed via API
func (r *RouterDoc) RegisterWithFiber(app *fiber.App, path string) {
	app.Get(path, func(c *fiber.Ctx) error {
		return c.JSON(r)
	})
}
