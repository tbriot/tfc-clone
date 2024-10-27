package main

type WorkspaceVariable struct {
	Id string
	// The name of the variable.
	Key   string
	Value string
	// Whether this is a Terraform or environment variable. Valid values are "terraform" or "env".
	Category string
	// Whether the value is sensitive. If true then the variable is written once and not visible thereafter.
	Sensitive bool
}

type TfcWorkspaceVariablesManager interface {
	ListWorkspaceVariables(workspaceId string) (*[]WorkspaceVariable, error)
}
