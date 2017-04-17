package tags_test

import (
	"testing"
	"github.com/jonbodner/proteus-talk/tags"
)

type S struct {
	field1 int `tag1:"hello" tag2:"goodbye"`
	Field2 int `tag1:"hola" tag2:"adiós"`
	Field3 string
}

func TestIt(t *testing.T) {
	tags.TagPrinter(S{})
}
