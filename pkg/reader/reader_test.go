package reader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRead(t *testing.T) {
	kr := KustomizeReader{}
	_, err := kr.Read("fixtures/nginx/base")

	assert.Equal(t, nil, err, nil)
}

func TestReadError(t *testing.T) {
	kr := KustomizeReader{}
	_, err := kr.Read("")

	assert.Error(t, err, nil)
}
