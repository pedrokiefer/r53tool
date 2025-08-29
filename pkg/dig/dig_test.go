package dig

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeResolver struct{ err error }

func (f fakeResolver) Resolve(ctx context.Context, domain string, t string) error {
	return f.err
}

func TestResolve_DelegatesToCurrentResolver(t *testing.T) {
	t.Cleanup(func() { CurrentResolver = realResolver{} })

	want := errors.New("boom")
	CurrentResolver = fakeResolver{err: want}

	got := Resolve(context.Background(), "example.com", "A")
	require.Equal(t, want, got)
}

func TestResolve_RealResolver_InvalidType(t *testing.T) {
	// Keep real resolver, check invalid type error
	err := realResolver{}.Resolve(context.Background(), "example.com", "INVALID")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid type")
}
