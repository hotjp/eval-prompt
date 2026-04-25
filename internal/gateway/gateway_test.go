package gateway

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	g := New()
	require.NotNil(t, g)
}

func TestGateway_Start(t *testing.T) {
	g := New()
	err := g.Start(context.Background())
	require.NoError(t, err)
}

func TestGateway_Stop(t *testing.T) {
	g := New()
	err := g.Stop(context.Background())
	require.NoError(t, err)
}
