package doubao

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeDoubaoBaseURL(t *testing.T) {
	assert.Equal(t, "https://ark.cn-beijing.volces.com", normalizeDoubaoBaseURL("https://ark.cn-beijing.volces.com"))
	assert.Equal(t, "https://ark.cn-beijing.volces.com", normalizeDoubaoBaseURL("https://ark.cn-beijing.volces.com/"))
	assert.Equal(t, "https://ark.cn-beijing.volces.com", normalizeDoubaoBaseURL("https://ark.cn-beijing.volces.com/api"))
	assert.Equal(t, "https://ark.cn-beijing.volces.com", normalizeDoubaoBaseURL("https://ark.cn-beijing.volces.com/api/"))
}
