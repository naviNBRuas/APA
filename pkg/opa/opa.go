package opa

import (
	"context"
	"fmt"
	"os"

	"github.com/open-policy-agent/opa/rego"
)

// OPAPolicyEngine manages OPA policy loading and evaluation.
type OPAPolicyEngine struct {
	query rego.PreparedEvalQuery
}

// NewOPAPolicyEngine creates a new OPA policy engine.
func NewOPAPolicyEngine() *OPAPolicyEngine {
	return &OPAPolicyEngine{}
}

// LoadPolicy loads a Rego policy from the given file path.
func (o *OPAPolicyEngine) LoadPolicy(ctx context.Context, policyPath string) error {
	policyBytes, err := os.ReadFile(policyPath)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	// Create a new Rego instance with the policy module
	r := rego.New(
		rego.Module("policy.rego", string(policyBytes)),
		rego.Query("data.apa.authz.allow"), // Assuming policy defines 'data.apa.authz.allow'
	)

	// Prepare the query for evaluation
	query, err := r.PrepareForEval(ctx)
	if err != nil {
		return fmt.Errorf("failed to prepare policy query: %w", err)
	}
	o.query = query

	return nil
}

// Authorize evaluates an authorization request against the loaded policy.
// input should be a map[string]interface{} representing the authorization context.
func (o *OPAPolicyEngine) Authorize(ctx context.Context, input map[string]interface{}) (bool, error) {

	resultSet, err := o.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false, fmt.Errorf("failed to evaluate policy: %w", err)
	}

	if len(resultSet) == 0 {
		// No result means the policy did not produce a decision for 'allow'
		return false, nil
	}

	// Assuming the policy returns a boolean 'allow' value
	allow, ok := resultSet[0].Expressions[0].Value.(bool)
	if !ok {
		return false, fmt.Errorf("policy did not return a boolean 'allow' value")
	}

	return allow, nil
}
