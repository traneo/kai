package validation

type ApprovalGate struct {
	required bool
}

func NewApprovalGate(required bool) *ApprovalGate {
	return &ApprovalGate{required: required}
}

func (g *ApprovalGate) Name() Type {
	return TypeApproval
}

func (g *ApprovalGate) Run(ctx *Context) *Result {
	if !g.required {
		return &Result{
			Gate:    TypeApproval,
			Status:  StatusSkipped,
			Message: "approval not required for this step",
		}
	}

	return &Result{
		Gate:    TypeApproval,
		Status:  StatusPending,
		Message: "waiting for human approval",
	}
}
