package documents

import (
	"cli/internal/dto"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type memoryDocs struct {
	input dto.DocumentInput
	err   error
	calls int
}

func (a *memoryDocs) CurrentDocuments(context.Context, int, string) ([]dto.Document, error) {
	return []dto.Document{{ID: 91, DocType: "notes", Body: "raw\tbytes\n"}}, a.err
}

func (a *memoryDocs) InsertDocument(_ context.Context, input dto.DocumentInput) (int, error) {
	a.input = input
	a.calls++
	return 92, a.err
}

func TestP406ConfiguredDocumentAccessAndExactBytes(t *testing.T) {
	a := &memoryDocs{}
	access := Access{API: a, Types: []string{"notes"}}
	docs, err := access.Load(context.Background(), 12)
	require.NoError(t, err)
	require.Equal(t, "raw\tbytes\n", docs[0].Body)
	id, err := access.Save(context.Background(), 12, "notes", docs[0].Body)
	require.NoError(t, err)
	require.Equal(t, 92, id)
	require.Equal(t, dto.DocumentInput{RefID: 12, RefTable: "change", DocType: "notes", Body: docs[0].Body, AgentEdit: false}, a.input)
	_, err = access.Save(context.Background(), 12, "brief", "text")
	require.ErrorContains(t, err, "not configured")
	_, err = access.Save(context.Background(), 12, "notes", " ")
	require.ErrorContains(t, err, "required")
	require.Equal(t, 1, a.calls)
	a.err = errors.New("save failed")
	_, err = access.Save(context.Background(), 12, "notes", "text")
	require.ErrorIs(t, err, a.err)
}
