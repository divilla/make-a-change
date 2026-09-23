package testcase

import (
	"context"
	"errors"

	"aipm/internal/dto"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo defines Repo values.
type Repo struct {
	pool *pgxpool.Pool
}

const changeColumns = `
	id,
	ref_uuid,
	ref,
	version,
	slug,
	project_id,
	change_phase,
	change_types,
	epic_id,
	epic_name,
	title,
	def,
	spec,
	pr,
	pr_url,
	agent_edit,
	open,
	done_tc,
	total_tc,
	completed,
	created,
	modified`

// NewRepo initializes or executes NewRepo behavior.
func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// List executes List behavior.
func (r *Repo) List(ctx context.Context, changeID int) ([]dto.TestCase, error) {
	if err := ensureChangeExists(ctx, r.pool, changeID); err != nil {
		return nil, err
	}
	return listTestCases(ctx, r.pool, changeID)
}

// Create executes Create behavior.
func (r *Repo) Create(ctx context.Context, req dto.TestCaseCreateRequest) (dto.TestCaseMutationResponse, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	defer tx.Rollback(ctx)

	if err := ensureChangeExists(ctx, tx, req.ChangeID); err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	testCase, err := scanTestCase(tx.QueryRow(ctx, `
		insert into public.test_case (scenario, change_id)
		values ($1, $2)
		returning id, version, scenario, done, change_id, created, modified
	`, req.Scenario, req.ChangeID))
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	return finishMutation(ctx, tx, testCase.ChangeID, []int{testCase.ChangeID}, &testCase)
}

// Update executes Update behavior.
func (r *Repo) Update(ctx context.Context, req dto.TestCaseUpdateRequest) (dto.TestCaseMutationResponse, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	defer tx.Rollback(ctx)

	current, err := getTestCase(ctx, tx, req.ID)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	if req.Scenario == current.Scenario {
		return finishMutation(ctx, tx, current.ChangeID, nil, &current)
	}
	if _, err := tx.Exec(ctx, "call public.sp_test_case_to_history($1, false)", req.ID); err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	testCase, err := scanTestCase(tx.QueryRow(ctx, `
		update public.test_case
		set scenario = $2,
			version = version + 1,
			modified = now()
		where id = $1
		returning id, version, scenario, done, change_id, created, modified
	`, req.ID, req.Scenario))
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.TestCaseMutationResponse{}, ErrNotFound
	}
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	return finishMutation(ctx, tx, current.ChangeID, nil, &testCase)
}

// UpdateDone executes UpdateDone behavior.
func (r *Repo) UpdateDone(ctx context.Context, req dto.TestCaseUpdateDoneRequest) (dto.TestCaseMutationResponse, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	defer tx.Rollback(ctx)

	current, err := getTestCase(ctx, tx, req.ID)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	if req.Done == current.Done {
		return finishMutation(ctx, tx, current.ChangeID, nil, &current)
	}
	testCase, err := scanTestCase(tx.QueryRow(ctx, `
		update public.test_case
		set done = $2,
			modified = now()
		where id = $1
		returning id, version, scenario, done, change_id, created, modified
	`, req.ID, req.Done))
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	return finishMutation(ctx, tx, current.ChangeID, []int{current.ChangeID}, &testCase)
}

// UpdateChange executes UpdateChange behavior.
func (r *Repo) UpdateChange(ctx context.Context, req dto.TestCaseUpdateChangeRequest) (dto.TestCaseMutationResponse, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	defer tx.Rollback(ctx)

	current, err := getTestCase(ctx, tx, req.ID)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	if err := ensureChangeExists(ctx, tx, req.ChangeID); err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	if req.ChangeID == current.ChangeID {
		return finishMutation(ctx, tx, current.ChangeID, nil, &current)
	}
	testCase, err := scanTestCase(tx.QueryRow(ctx, `
		update public.test_case
		set change_id = $2,
			modified = now()
		where id = $1
		returning id, version, scenario, done, change_id, created, modified
	`, req.ID, req.ChangeID))
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	return finishMutation(ctx, tx, req.ChangeID, []int{current.ChangeID, req.ChangeID}, &testCase)
}

// Delete executes Delete behavior.
func (r *Repo) Delete(ctx context.Context, req dto.TestCaseIDRequest) (dto.TestCaseMutationResponse, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	defer tx.Rollback(ctx)

	current, err := getTestCase(ctx, tx, req.ID)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	if _, err := tx.Exec(ctx, "call public.sp_test_case_to_history($1, true)", req.ID); err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	tag, err := tx.Exec(ctx, "delete from public.test_case where id = $1", req.ID)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	if tag.RowsAffected() == 0 {
		return dto.TestCaseMutationResponse{}, ErrNotFound
	}
	return finishMutation(ctx, tx, current.ChangeID, []int{current.ChangeID}, nil)
}

func finishMutation(ctx context.Context, tx pgx.Tx, responseChangeID int, affectedChangeIDs []int, testCase *dto.TestCase) (dto.TestCaseMutationResponse, error) {
	for _, changeID := range uniqueIDs(affectedChangeIDs) {
		if _, err := tx.Exec(ctx, "call public.sp_change_test_case_recalculate($1)", changeID); err != nil {
			return dto.TestCaseMutationResponse{}, err
		}
	}
	change, err := getChange(ctx, tx, responseChangeID)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	testCases, err := listTestCases(ctx, tx, responseChangeID)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	return dto.TestCaseMutationResponse{TestCase: testCase, Change: change, TestCases: testCases}, nil
}

func getTestCase(ctx context.Context, tx pgx.Tx, id int) (dto.TestCase, error) {
	testCase, err := scanTestCase(tx.QueryRow(ctx, `
		select id, version, scenario, done, change_id, created, modified
		from public.test_case
		where id = $1
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.TestCase{}, ErrNotFound
	}
	if err != nil {
		return dto.TestCase{}, err
	}
	return testCase, nil
}

func ensureChangeExists(ctx context.Context, q queryer, id int) error {
	var exists bool
	if err := q.QueryRow(ctx, "select exists(select 1 from public.change where id = $1)", id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

func listTestCases(ctx context.Context, q queryer, changeID int) ([]dto.TestCase, error) {
	rows, err := q.Query(ctx, `
		select id, version, scenario, done, change_id, created, modified
		from public.test_case
		where change_id = $1
		order by id
	`, changeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	testCases := make([]dto.TestCase, 0)
	for rows.Next() {
		testCase, err := scanTestCase(rows)
		if err != nil {
			return nil, err
		}
		testCases = append(testCases, testCase)
	}
	return testCases, rows.Err()
}

func scanTestCase(row pgx.Row) (dto.TestCase, error) {
	var testCase dto.TestCase
	err := row.Scan(
		&testCase.ID, &testCase.Version, &testCase.Scenario, &testCase.Done,
		&testCase.ChangeID, &testCase.Created, &testCase.Modified,
	)
	return testCase, err
}

func getChange(ctx context.Context, q queryer, id int) (dto.Change, error) {
	change, err := scanChange(q.QueryRow(ctx, "select "+changeColumns+" from public.vw_change_details where id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Change{}, ErrNotFound
	}
	return change, err
}

func scanChange(row pgx.Row) (dto.Change, error) {
	var change dto.Change
	var refUUID pgtype.UUID
	var ref pgtype.Int4
	var slug pgtype.Text
	var epicID pgtype.Int8
	var epicName pgtype.Text
	err := row.Scan(
		&change.ID, &refUUID, &ref, &change.Version, &slug, &change.ProjectID,
		&change.ChangePhase, &change.ChangeTypes, &epicID, &epicName, &change.Title,
		&change.Def, &change.Spec, &change.PR, &change.PRUrl, &change.AgentEdit, &change.Open, &change.DoneTC,
		&change.TotalTC, &change.Completed, &change.Created, &change.Modified,
	)
	if err != nil {
		return dto.Change{}, err
	}
	if refUUID.Valid {
		change.RefUUID = refUUID.String()
	}
	if ref.Valid {
		value := ref.Int32
		change.Ref = &value
	}
	if slug.Valid {
		value := slug.String
		change.Slug = &value
	}
	if epicID.Valid {
		value := int(epicID.Int64)
		change.EpicID = &value
	}
	if epicName.Valid {
		value := epicName.String
		change.EpicName = &value
	}
	return change, nil
}

func uniqueIDs(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}
