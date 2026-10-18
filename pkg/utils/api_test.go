package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetContentDispositionOfAttachment_AsciiFileName(t *testing.T) {
	assert.Equal(t, "attachment;filename=user_2026_10_18.csv", GetContentDispositionOfAttachment("user_2026_10_18.csv"))
}

func TestGetContentDispositionOfAttachment_NonAsciiFileName(t *testing.T) {
	assert.Equal(t, "attachment;filename=user_book____2026.csv;filename*=UTF-8''user_book_%E5%AE%B6%E5%BA%AD_2026.csv", GetContentDispositionOfAttachment("user_book_家庭_2026.csv"))
}
