package resources

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	tb "github.com/tektoncd/pipeline/internal/builder/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/diff"
	"k8s.io/apimachinery/pkg/selection"
)

func TestTaskParamResolver_ResolveResultRefs(t *testing.T) {
	for _, tt := range []struct {
		name             string
		pipelineRunState PipelineRunState
		param            v1beta1.Param
		want             ResolvedResultRefs
		wantErr          bool
	}{{
		name: "successful resolution: param not using result reference",
		pipelineRunState: PipelineRunState{{
			TaskRunName: "aTaskRun",
			TaskRun:     tb.TaskRun("aTaskRun"),
			PipelineTask: &v1beta1.PipelineTask{
				Name:    "aTask",
				TaskRef: &v1beta1.TaskRef{Name: "aTask"},
			},
		}},
		param: v1beta1.Param{
			Name:  "targetParam",
			Value: *v1beta1.NewArrayOrString("explicitValueNoResultReference"),
		},
		want:    nil,
		wantErr: false,
	}, {
		name: "successful resolution: using result reference",
		pipelineRunState: PipelineRunState{{
			TaskRunName: "aTaskRun",
			TaskRun: tb.TaskRun("aTaskRun", tb.TaskRunStatus(
				tb.TaskRunResult("aResult", "aResultValue"),
			)),
			PipelineTask: &v1beta1.PipelineTask{
				Name:    "aTask",
				TaskRef: &v1beta1.TaskRef{Name: "aTask"},
			},
		}},
		param: v1beta1.Param{
			Name:  "targetParam",
			Value: *v1beta1.NewArrayOrString("$(tasks.aTask.results.aResult)"),
		},
		want: ResolvedResultRefs{{
			Value: *v1beta1.NewArrayOrString("aResultValue"),
			ResultReference: v1beta1.ResultRef{
				PipelineTask: "aTask",
				Result:       "aResult",
			},
			FromTaskRun: "aTaskRun",
		}},
		wantErr: false,
	}, {
		name: "successful resolution: using multiple result reference",
		pipelineRunState: PipelineRunState{{
			TaskRunName: "aTaskRun",
			TaskRun: tb.TaskRun("aTaskRun", tb.TaskRunStatus(
				tb.TaskRunResult("aResult", "aResultValue"),
			)),
			PipelineTask: &v1beta1.PipelineTask{
				Name:    "aTask",
				TaskRef: &v1beta1.TaskRef{Name: "aTask"},
			},
		}, {
			TaskRunName: "bTaskRun",
			TaskRun: tb.TaskRun("bTaskRun", tb.TaskRunStatus(
				tb.TaskRunResult("bResult", "bResultValue"),
			)),
			PipelineTask: &v1beta1.PipelineTask{
				Name:    "bTask",
				TaskRef: &v1beta1.TaskRef{Name: "bTask"},
			},
		}},
		param: v1beta1.Param{
			Name:  "targetParam",
			Value: *v1beta1.NewArrayOrString("$(tasks.aTask.results.aResult) $(tasks.bTask.results.bResult)"),
		},
		want: ResolvedResultRefs{{
			Value: *v1beta1.NewArrayOrString("aResultValue"),
			ResultReference: v1beta1.ResultRef{
				PipelineTask: "aTask",
				Result:       "aResult",
			},
			FromTaskRun: "aTaskRun",
		}, {
			Value: *v1beta1.NewArrayOrString("bResultValue"),
			ResultReference: v1beta1.ResultRef{
				PipelineTask: "bTask",
				Result:       "bResult",
			},
			FromTaskRun: "bTaskRun",
		}},
		wantErr: false,
	}, {
		name: "successful resolution: duplicate result references",
		pipelineRunState: PipelineRunState{{
			TaskRunName: "aTaskRun",
			TaskRun: tb.TaskRun("aTaskRun", tb.TaskRunStatus(
				tb.TaskRunResult("aResult", "aResultValue"),
			)),
			PipelineTask: &v1beta1.PipelineTask{
				Name:    "aTask",
				TaskRef: &v1beta1.TaskRef{Name: "aTask"},
			},
		}},
		param: v1beta1.Param{
			Name:  "targetParam",
			Value: *v1beta1.NewArrayOrString("$(tasks.aTask.results.aResult) $(tasks.aTask.results.aResult)"),
		},
		want: ResolvedResultRefs{{
			Value: *v1beta1.NewArrayOrString("aResultValue"),
			ResultReference: v1beta1.ResultRef{
				PipelineTask: "aTask",
				Result:       "aResult",
			},
			FromTaskRun: "aTaskRun",
		}},
		wantErr: false,
	}, {
		name: "unsuccessful resolution: referenced result doesn't exist in referenced task",
		pipelineRunState: PipelineRunState{{
			TaskRunName: "aTaskRun",
			TaskRun:     tb.TaskRun("aTaskRun"),
			PipelineTask: &v1beta1.PipelineTask{
				Name:    "aTask",
				TaskRef: &v1beta1.TaskRef{Name: "aTask"},
			},
		}},
		param: v1beta1.Param{
			Name:  "targetParam",
			Value: *v1beta1.NewArrayOrString("$(tasks.aTask.results.aResult)"),
		},
		want:    nil,
		wantErr: true,
	}, {
		name:             "unsuccessful resolution: pipeline task missing",
		pipelineRunState: PipelineRunState{},
		param: v1beta1.Param{
			Name:  "targetParam",
			Value: *v1beta1.NewArrayOrString("$(tasks.aTask.results.aResult)"),
		},
		want:    nil,
		wantErr: true,
	}, {
		name: "unsuccessful resolution: task run missing",
		pipelineRunState: PipelineRunState{{
			PipelineTask: &v1beta1.PipelineTask{
				Name:    "aTask",
				TaskRef: &v1beta1.TaskRef{Name: "aTask"},
			},
		}},
		param: v1beta1.Param{
			Name:  "targetParam",
			Value: *v1beta1.NewArrayOrString("$(tasks.aTask.results.aResult)"),
		},
		want:    nil,
		wantErr: true,
	}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("test name: %s\n", tt.name)
			got, err := extractResultRefsForParam(tt.pipelineRunState, tt.param)
			// sort result ref based on task name to guarantee an certain order
			sort.SliceStable(got, func(i, j int) bool {
				return strings.Compare(got[i].FromTaskRun, got[j].FromTaskRun) < 0
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolveResultRef() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(tt.want) != len(got) {
				t.Fatalf("incorrect number of refs, want %d, got %d", len(tt.want), len(got))
			}
			for _, rGot := range got {
				foundMatch := false
				for _, rWant := range tt.want {
					if d := cmp.Diff(rGot, rWant); d == "" {
						foundMatch = true
					}
				}
				if !foundMatch {
					t.Fatalf("Expected resolved refs:\n%s\n\nbut received:\n%s\n", resolvedSliceAsString(tt.want), resolvedSliceAsString(got))
				}
			}
		})
	}
}

func resolvedSliceAsString(rs []*ResolvedResultRef) string {
	var s []string
	for _, r := range rs {
		s = append(s, fmt.Sprintf("%#v", *r))
	}
	return fmt.Sprintf("[\n%s\n]", strings.Join(s, ",\n"))
}

func TestResolveResultRefs(t *testing.T) {
	pipelineRunState := PipelineRunState{{
		TaskRunName: "aTaskRun",
		TaskRun: tb.TaskRun("aTaskRun", tb.TaskRunStatus(
			tb.TaskRunResult("aResult", "aResultValue"),
		)),
		PipelineTask: &v1beta1.PipelineTask{
			Name:    "aTask",
			TaskRef: &v1beta1.TaskRef{Name: "aTask"},
		},
	}, {
		PipelineTask: &v1beta1.PipelineTask{
			Name:    "bTask",
			TaskRef: &v1beta1.TaskRef{Name: "bTask"},
			Params: []v1beta1.Param{{
				Name:  "bParam",
				Value: *v1beta1.NewArrayOrString("$(tasks.aTask.results.aResult)"),
			}},
		},
	}, {
		PipelineTask: &v1beta1.PipelineTask{
			Name:    "bTask",
			TaskRef: &v1beta1.TaskRef{Name: "bTask"},
			WhenExpressions: []v1beta1.WhenExpression{{
				Input:    "$(tasks.aTask.results.aResult)",
				Operator: selection.In,
				Values:   []string{"$(tasks.aTask.results.aResult)"},
			}},
		},
	}, {
		PipelineTask: &v1beta1.PipelineTask{
			Name:    "bTask",
			TaskRef: &v1beta1.TaskRef{Name: "bTask"},
			WhenExpressions: []v1beta1.WhenExpression{{
				Input:    "$(tasks.aTask.results.missingResult)",
				Operator: selection.In,
				Values:   []string{"$(tasks.aTask.results.missingResult)"},
			}},
		},
	}, {
		PipelineTask: &v1beta1.PipelineTask{
			Name:    "bTask",
			TaskRef: &v1beta1.TaskRef{Name: "bTask"},
			Params: []v1beta1.Param{{
				Name:  "bParam",
				Value: *v1beta1.NewArrayOrString("$(tasks.aTask.results.missingResult)"),
			}},
		},
	}}

	for _, tt := range []struct {
		name             string
		pipelineRunState PipelineRunState
		targets          PipelineRunState
		want             ResolvedResultRefs
		wantErr          bool
	}{{
		name:             "Test successful result references resolution - params",
		pipelineRunState: pipelineRunState,
		targets: PipelineRunState{
			pipelineRunState[1],
		},
		want: ResolvedResultRefs{{
			Value: *v1beta1.NewArrayOrString("aResultValue"),
			ResultReference: v1beta1.ResultRef{
				PipelineTask: "aTask",
				Result:       "aResult",
			},
			FromTaskRun: "aTaskRun",
		}},
		wantErr: false,
	}, {
		name:             "Test successful result references resolution - when expressions",
		pipelineRunState: pipelineRunState,
		targets: PipelineRunState{
			pipelineRunState[2],
		},
		want: ResolvedResultRefs{{
			Value: *v1beta1.NewArrayOrString("aResultValue"),
			ResultReference: v1beta1.ResultRef{
				PipelineTask: "aTask",
				Result:       "aResult",
			},
			FromTaskRun: "aTaskRun",
		}},
		wantErr: false,
	}, {
		name:             "Test successful result references resolution non result references",
		pipelineRunState: pipelineRunState,
		targets: PipelineRunState{
			pipelineRunState[0],
		},
		want:    nil,
		wantErr: false,
	}, {
		name:             "Test unsuccessful result references resolution - when expression",
		pipelineRunState: pipelineRunState,
		targets: PipelineRunState{
			pipelineRunState[3],
		},
		want:    nil,
		wantErr: true,
	}, {
		name:             "Test unsuccessful result references resolution - params",
		pipelineRunState: pipelineRunState,
		targets: PipelineRunState{
			pipelineRunState[4],
		},
		want:    nil,
		wantErr: true,
	}} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveResultRefs(tt.pipelineRunState, tt.targets)
			sort.SliceStable(got, func(i, j int) bool {
				return strings.Compare(got[i].FromTaskRun, got[j].FromTaskRun) < 0
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveResultRefs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if d := cmp.Diff(tt.want, got); d != "" {
				t.Fatalf("ResolveResultRef %s", diff.PrintWantGot(d))
			}
		})
	}
}
