package replication

type Role string

const (
	RolePrimary Role = "primary"
	RoleReplica Role = "replica"
)
