package model

type Document struct {
	ID string `json:"id"`

	Title string `json:"title"`

	Content string `json:"content"`

	Category string `json:"category"`

	Tags []string `json:"tags"`

	Format string `json:"format"`

	UploadTime int64 `json:"upload_time"`

	UpdateTime int64 `json:"update_time"`

	FileSize int64 `json:"file_size"`

	Checksum string `json:"checksum"`
}

func (d *Document) Normalize() {
	if d.Format == "" {
		d.Format = "txt"
	}
	if d.Tags == nil {
		d.Tags = []string{}
	}
}

func (d *Document) HasTag(name string) bool {
	for _, t := range d.Tags {
		if t == name {
			return true
		}
	}
	return false
}

func (d *Document) AddTag(name string) {
	if name == "" || d.HasTag(name) {
		return
	}
	d.Tags = append(d.Tags, name)
}

func (d *Document) RemoveTag(name string) {
	out := d.Tags[:0]
	for _, t := range d.Tags {
		if t != name {
			out = append(out, t)
		}
	}
	d.Tags = out
}
