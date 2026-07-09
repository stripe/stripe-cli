package coopcmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func TestRecentSessionChoicesSortsMostRecentFirst(t *testing.T) {
	store, err := coop.NewStoreAt(t.TempDir())
	require.NoError(t, err)

	require.NoError(t, store.Write(&coop.Session{
		ID:        "a_old",
		Blueprint: "old-blueprint",
		Status:    coop.SessionActive,
	}))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, store.Write(&coop.Session{
		ID:        "z_new",
		Blueprint: "new-blueprint",
		Status:    coop.SessionActive,
	}))

	choices, err := recentSessionChoices(store)

	require.NoError(t, err)
	require.Len(t, choices, 2)
	assert.Equal(t, "z_new", choices[0].session.ID)
	assert.Equal(t, "a_old", choices[1].session.ID)
}
