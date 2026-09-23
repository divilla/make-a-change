package changes

import "cli/internal/dto"

// API defines backend operations needed by change screens.
type API interface {
    ListChangeRows(projectID string) ([]dto.Change, error)
    GetChange(id int) (dto.Change, error)
    CreateChange(input dto.ChangeCreateInput) (dto.Change, error)
    ReferenceChange(id int) (dto.Change, error)
    UpdateChangeTitle(id int, title string) (dto.Change, error)
    UpdateChangeDef(id int, def string, agentEdit bool) (dto.Change, error)
    UpdateChangeSpec(id int, spec string, agentEdit bool) (dto.Change, error)
    UpdateChangePR(id int, pr string, agentEdit bool) (dto.Change, error)
    UpdateChangePRUrl(id int, prURL string) (dto.Change, error)
    UpdateChangeTypes(id int, changeTypes []string) (dto.Change, error)
    UpdateChangePhase(id int, changePhase string) (dto.Change, error)
    UpdateChangeOpen(id int, open bool) (dto.Change, error)
    UpdateChangeEpic(id int, epicID *int) (dto.Change, error)
    CreateTestCase(changeID int, scenario string) (dto.Change, error)
    UpdateTestCase(id int, scenario string) (dto.Change, error)
    UpdateTestCaseDone(id int, done bool) (dto.Change, error)
    DeleteTestCase(id int) (dto.Change, error)
    DeleteChange(id int) error
    ListPhases() ([]dto.Option, error)
}
