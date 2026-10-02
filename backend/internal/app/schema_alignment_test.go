package app

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsErrorPreservesWrappedSemantics(t *testing.T) {
	require.True(t, IsError(WrapError(ErrProjectHasChanges, "delete"), ErrProjectHasChanges))
	require.False(t, IsError(errors.New("unrelated database failure"), ErrProjectHasChanges))
	require.False(t, IsError(nil, ErrEpicHasChanges))
	require.True(t, IsError(nil, nil))
}
