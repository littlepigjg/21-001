package model

type Category struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	CreateTime int64 `json:"create_time"`
}

func (c *Category) Normalize() {
	if c.Description == "" {
		c.Description = ""
	}
}
