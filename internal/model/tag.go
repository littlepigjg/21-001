package model

type Tag struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Count int `json:"count"`

	CreateTime int64 `json:"create_time"`
}

func (t *Tag) Normalize() {
	if t.Count < 0 {
		t.Count = 0
	}
}
