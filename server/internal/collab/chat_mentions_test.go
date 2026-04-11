package collab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractChatMentions(t *testing.T) {
	assert.Nil(t, ExtractChatMentions("", 10))
	assert.Nil(t, ExtractChatMentions("hello", 10))
	assert.Equal(t, []string{"alice"}, ExtractChatMentions("hi @alice", 10))
	assert.Equal(t, []string{"alice", "bob"}, ExtractChatMentions("@alice and @bob @alice", 10))
	assert.Equal(t, []string{"a", "b"}, ExtractChatMentions("@a @b @c", 2))
}
