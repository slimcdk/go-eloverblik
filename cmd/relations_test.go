package cmd

import (
	"strings"
	"testing"

	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// retiredRelationsClient fails the test if a command calls one of the endpoints Energinet
// retired: the answer is known, so asking the API again is wasted traffic.
type retiredRelationsClient struct {
	MockCustomerClient
	t *testing.T
}

func (m *retiredRelationsClient) AddRelationByWebAccessCode(string, string) (string, error) {
	m.t.Error("add-relation-by-code called the API")
	return "", nil
}

func (m *retiredRelationsClient) DeleteRelation(string) (bool, error) {
	m.t.Error("delete-relation called the API")
	return false, nil
}

// TestRetiredRelationCommands covers the two commands whose endpoints Energinet retired
// with DataHub 3.0. They must say so without calling the API, and the help must no longer
// offer them: the root's Long text names them only to say they are retired, and the
// command listing leaves them out.
func TestRetiredRelationCommands(t *testing.T) {
	clientInstance = &retiredRelationsClient{t: t}
	defer func() { clientInstance = nil }()

	_, err := execute(t, "customer", "add-relation-by-code", "571313174002485069", "ABCD1234", "--token", "dummy")
	require.ErrorIs(t, err, eloverblik.ErrorEndpointRetired)

	_, err = execute(t, "customer", "delete-relation", "571313174002485069", "--token", "dummy")
	require.ErrorIs(t, err, eloverblik.ErrorEndpointRetired)

	help, err := execute(t, "--help")
	require.NoError(t, err)
	assert.Contains(t, help, "add-relation-by-code and delete-relation are retired")

	_, commands, found := strings.Cut(help, "Available Commands:")
	require.True(t, found, "the root help lists the commands")
	assert.NotContains(t, commands, "add-relation-by-code")
	assert.NotContains(t, commands, "delete-relation")
	assert.Contains(t, commands, "add-relation", "the relation command that still works must stay")
}
