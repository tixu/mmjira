package renderer

import "bytes"

// Renderer specifies the rendering behavior
type Renderer interface {
	Render() (text *bytes.Buffer, err error)
}
