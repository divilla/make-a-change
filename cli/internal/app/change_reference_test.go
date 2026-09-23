package app

import (
    "context"
    "errors"
    "strings"
    "testing"

    "cli/internal/dto"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

type recordedGitCall struct {
    dir  string
    args []string
}

type fakeGitRunner struct {
    outputs map[string]string
    errs    map[string]error
    calls   []recordedGitCall
}

func (f *fakeGitRunner) run(ctx context.Context, dir string, args ...string) (string, error) {
    f.calls = append(f.calls, recordedGitCall{dir: dir, args: append([]string(nil), args...)})
    key := strings.Join(args, "\x00")
    if err := f.errs[key]; err != nil {
        return f.outputs[key], err
    }
    return f.outputs[key], nil
}

func TestChangeReferenceCommandReferencesBranchAndReloads(t *testing.T) {
    originalRunner := referenceGitRunner
    defer func() { referenceGitRunner = originalRunner }()

    var gitCalls [][]string
    referenceGitRunner = func(ctx context.Context, dir string, args ...string) (string, error) {
        gitCalls = append(gitCalls, append([]string(nil), args...))
        switch strings.Join(args, " ") {
        case "rev-parse --show-toplevel":
            return "/repo", nil
        case "branch --list --format=%(refname:short) change/3-new-change":
            return "change/3-new-change", nil
        case "checkout change/3-new-change":
            return "", nil
        default:
            return "", nil
        }
    }

    client := &fakeClient{
        gotChange: dto.Change{ID: "12", Ref: "3", Slug: "3-new-change", Title: "New Change"},
    }
    m := NewModelWithClient(client)
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{ID: "12", Title: "New Change"})

    got, cmd := sendCommand(m, "/reference")
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, dto.Change{ID: "12", Ref: "3", Slug: "3-new-change", Title: "New Change"}, got.changeList.Detail)
    assert.Equal(t, []int{12}, client.changeReferenceIDs)
    assert.Equal(t, []int{12}, client.changeGetIDs)
    assert.Contains(t, gitCalls, []string{"checkout", "change/3-new-change"})
    assert.Empty(t, got.err)
}

func TestChangeReferenceCommandBackendFailureSkipsGit(t *testing.T) {
    originalRunner := referenceGitRunner
    defer func() { referenceGitRunner = originalRunner }()

    var gitCalls int
    referenceGitRunner = func(ctx context.Context, dir string, args ...string) (string, error) {
        gitCalls++
        return "", nil
    }
    client := &fakeClient{changeReferenceErr: errors.New("reference failed")}
    m := NewModelWithClient(client)
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{ID: "12", Title: "New Change"})

    got, cmd := sendCommand(m, "/reference")
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, "reference failed", got.err)
    assert.Equal(t, 0, gitCalls)
    assert.Zero(t, client.changeGetCalls)
}

func TestChangeReferenceCommandGitFailureKeepsReferencedChangeLoaded(t *testing.T) {
    originalRunner := referenceGitRunner
    defer func() { referenceGitRunner = originalRunner }()

    referenceGitRunner = func(ctx context.Context, dir string, args ...string) (string, error) {
        switch strings.Join(args, " ") {
        case "rev-parse --show-toplevel":
            return "/repo", nil
        case "checkout -b change/3-new-change":
            return "", errors.New("checkout failed")
        default:
            return "", nil
        }
    }
    client := &fakeClient{
        gotChange: dto.Change{ID: "12", Ref: "3", Slug: "3-new-change", Title: "Referenced Change"},
    }
    m := NewModelWithClient(client)
    m.state = ChangeDetailsState
    m.changeList = m.changeList.WithDetail(dto.Change{ID: "12", Title: "Draft Change"})
    m.changeList.DetailSelected = 3

    got, cmd := sendCommand(m, "/reference")
    require.NotNil(t, cmd)
    got = applyMsg(got, cmd())

    assert.Equal(t, ChangeDetailsState, got.state)
    assert.Equal(t, "checkout failed", got.err)
    assert.Equal(t, dto.Change{ID: "12", Ref: "3", Slug: "3-new-change", Title: "Referenced Change"}, got.changeList.Detail)
    assert.Equal(t, 3, got.changeList.DetailSelected)
    assert.Equal(t, 1, client.changeGetCalls)
}

func TestReconcileChangeBranchPaths(t *testing.T) {
    tests := []struct {
        name    string
        outputs map[string]string
        want    [][]string
    }{
        {
            name: "local exact",
            outputs: map[string]string{
                "rev-parse\x00--show-toplevel":                                      "/repo",
                "branch\x00--list\x00--format=%(refname:short)\x00change/003-title": "change/003-title",
            },
            want: [][]string{
                {"checkout", "change/003-title"},
            },
        },
        {
            name: "local padded prefix rename",
            outputs: map[string]string{
                "rev-parse\x00--show-toplevel":                              "/repo",
                "branch\x00--list\x00--format=%(refname:short)\x00change/*": "change/030-wrong-title\nchange/003-old-title",
            },
            want: [][]string{
                {"checkout", "change/003-old-title"},
                {"branch", "-m", "change/003-title"},
            },
        },
        {
            name: "local raw prefix fallback rename",
            outputs: map[string]string{
                "rev-parse\x00--show-toplevel":                              "/repo",
                "branch\x00--list\x00--format=%(refname:short)\x00change/*": "change/3-old-title",
            },
            want: [][]string{
                {"checkout", "change/3-old-title"},
                {"branch", "-m", "change/003-title"},
            },
        },
        {
            name: "remote exact",
            outputs: map[string]string{
                "rev-parse\x00--show-toplevel":                                  "/repo",
                "ls-remote\x00--heads\x00origin\x00refs/heads/change/003-title": "abc refs/heads/change/003-title",
            },
            want: [][]string{
                {"fetch", "origin", "refs/heads/change/003-title:refs/remotes/origin/change/003-title"},
                {"checkout", "-b", "change/003-title", "--track", "origin/change/003-title"},
            },
        },
        {
            name: "remote padded prefix rename",
            outputs: map[string]string{
                "rev-parse\x00--show-toplevel":                          "/repo",
                "ls-remote\x00--heads\x00origin\x00refs/heads/change/*": "abc refs/heads/change/030-wrong-title\ndef refs/heads/change/003-old-title",
            },
            want: [][]string{
                {"fetch", "origin", "refs/heads/change/003-old-title:refs/remotes/origin/change/003-old-title"},
                {"checkout", "-b", "change/003-old-title", "--track", "origin/change/003-old-title"},
                {"branch", "-m", "change/003-title"},
                {"push", "origin", "change/003-title"},
                {"push", "origin", "--delete", "change/003-old-title"},
            },
        },
        {
            name: "remote raw prefix fallback rename",
            outputs: map[string]string{
                "rev-parse\x00--show-toplevel":                          "/repo",
                "ls-remote\x00--heads\x00origin\x00refs/heads/change/*": "abc refs/heads/change/3-old-title",
            },
            want: [][]string{
                {"fetch", "origin", "refs/heads/change/3-old-title:refs/remotes/origin/change/3-old-title"},
                {"checkout", "-b", "change/3-old-title", "--track", "origin/change/3-old-title"},
                {"branch", "-m", "change/003-title"},
                {"push", "origin", "change/003-title"},
                {"push", "origin", "--delete", "change/3-old-title"},
            },
        },
        {
            name: "create new",
            outputs: map[string]string{
                "rev-parse\x00--show-toplevel": "/repo",
            },
            want: [][]string{
                {"checkout", "-b", "change/003-title"},
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            runner := &fakeGitRunner{outputs: tt.outputs, errs: map[string]error{}}

            err := reconcileChangeBranchWithRunner(context.Background(), runner.run, "3", "003-title")
            require.NoError(t, err)

            var got [][]string
            for _, call := range runner.calls {
                if len(call.args) > 0 && call.args[0] == "rev-parse" || len(call.args) >= 2 && call.args[0] == "branch" && call.args[1] == "--list" || len(call.args) > 0 && call.args[0] == "ls-remote" {
                    continue
                }
                got = append(got, call.args)
            }
            assert.Equal(t, tt.want, got)
        })
    }
}

func TestReconcileChangeBranchRemoteFetchFailureSkipsCheckout(t *testing.T) {
    runner := &fakeGitRunner{
        outputs: map[string]string{
            "rev-parse\x00--show-toplevel":                                  "/repo",
            "ls-remote\x00--heads\x00origin\x00refs/heads/change/003-title": "abc refs/heads/change/003-title",
        },
        errs: map[string]error{
            "fetch\x00origin\x00refs/heads/change/003-title:refs/remotes/origin/change/003-title": errors.New("fetch failed"),
        },
    }

    err := reconcileChangeBranchWithRunner(context.Background(), runner.run, "3", "003-title")
    require.EqualError(t, err, "fetch failed")

    for _, call := range runner.calls {
        require.NotEqual(t, "checkout", call.args[0])
    }
}
