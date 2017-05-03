package httplab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplit(t *testing.T) {
	t.Run("Fixed", func(t *testing.T) {
		/*
			| 1 2 3 4 5 6 7 8 9 A |
			| _ _ x _ _ x _ x _ _ |
		*/
		split := NewSplit(10).Fixed(3, 3, 2)
		assert.Equal(t, 3, split.Next())
		assert.Equal(t, 6, split.Next())
		assert.Equal(t, 8, split.Next())
		assert.Equal(t, 0, split.Next())
	})

	t.Run("Relative", func(t *testing.T) {

		/*
			| 1 2 3 4 5 6 7 8 9 A |
			| x _ _ _ _ x _ _ x _ |
		*/

		split := NewSplit(10).Relative(20, 50, 50)
		assert.Equal(t, 2, split.Next())
		assert.Equal(t, 6, split.Next())
		assert.Equal(t, 8, split.Next())
		assert.Equal(t, 0, split.Next())
	})

	/*
		| 1 2 3 4 5 6 7 8 9 A |
		| f f _ _ _ r f _ r _ |
	*/

	split := NewSplit(10).Fixed(1, 1).Relative(50).Fixed(1).Relative(50)
	assert.Equal(t, 1, split.Next())
	assert.Equal(t, 2, split.Next())
	assert.Equal(t, 6, split.Next())
	assert.Equal(t, 7, split.Next())
	assert.Equal(t, 9, split.Next())
	assert.Equal(t, 0, split.Next())

}
