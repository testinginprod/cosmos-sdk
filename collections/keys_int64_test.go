package collections

import "testing"

func TestInt64Key(t *testing.T) {
	t.Run("bijective1", func(t *testing.T) {
		assertBijective(t, Int64KeyEncoder, 1)
	})
	t.Run("bijectiv2", func(t *testing.T) {
		assertBijective(t, Int64KeyEncoder, -1)
	})
}
