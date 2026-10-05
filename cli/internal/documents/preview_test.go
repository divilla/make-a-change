package documents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreviewValidationPrinterFailureEmptyAndCancellation(t *testing.T) {
	h, cmd := (History{}).PreviewBody(context.Background(), nil, 0, 12, "change", "brief", "")
	require.Nil(t, cmd)
	require.Error(t, h.Err)
	h, cmd = h.PreviewBody(context.Background(), nil, 7, 12, "change", "brief", "")
	h = finishHistory(t, h, cmd)
	require.ErrorContains(t, h.Err, "printer unavailable")
	printer := &coloredPrinter{}
	h, cmd = h.PreviewBody(context.Background(), printer, 7, 12, "change", "brief", "")
	require.Equal(t, "Loading view…", h.View(80, 10))
	h = h.Invalidate()
	result := cmd().(HistoryResult)
	require.ErrorIs(t, result.Err, context.Canceled)
	_, ok := h.Apply(result)
	require.False(t, ok)
	require.Equal(t, "Empty item", h.View(80, 10))
}
