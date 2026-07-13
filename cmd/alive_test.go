package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestAliveCmdPerParent guards against sharing a single "alive" command between the two
// API parents. cobra sets the parent on AddCommand, so the last parent used to win and
// "customer alive" ran the ThirdParty PersistentPreRunE, probing the ThirdParty API.
func TestAliveCmdPerParent(t *testing.T) {
	parents := map[string]*cobra.Command{
		"customer":   customerCmd,
		"thirdparty": thirdpartyCmd,
	}

	found := make(map[string]*cobra.Command, len(parents))

	for name, parent := range parents {
		t.Run(name+" alive resolves to a command owned by its own parent", func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{name, "alive"})
			assert.NoError(t, err)
			assert.Equal(t, "alive", cmd.Name())
			assert.Same(t, parent, cmd.Parent(), "the alive command must belong to the %s command", name)

			found[name] = cmd
		})
	}

	t.Run("each parent has its own instance", func(t *testing.T) {
		assert.NotSame(t, found["customer"], found["thirdparty"])
	})
}
