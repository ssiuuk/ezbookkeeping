package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFileNameSafeText(t *testing.T) {
	assert.Equal(t, "Family_Book", getFileNameSafeText("  Family Book "))
	assert.Equal(t, "a_b_c_d_e", getFileNameSafeText("a/b\\c:d*e"))
	assert.Equal(t, "我的家庭", getFileNameSafeText("我的家庭"))
	assert.Equal(t, "", getFileNameSafeText("   "))
}

func TestGetBookMemberSubmissionId(t *testing.T) {
	assert.Equal(t, "", getBookMemberSubmissionId(100, 1, ""))
	assert.Equal(t, "session", getBookMemberSubmissionId(1, 1, "session"))
	assert.Equal(t, "1|session", getBookMemberSubmissionId(100, 1, "session"))
	assert.NotEqual(t, getBookMemberSubmissionId(100, 1, "session"), getBookMemberSubmissionId(100, 2, "session"))
}
