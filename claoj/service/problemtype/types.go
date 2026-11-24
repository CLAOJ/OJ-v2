// Package problemtype provides problem type management services.
package problemtype

// ProblemType represents a problem type.
type ProblemType struct {
	ID           uint
	Name         string
	FullName     string
	ProblemCount int64
	ProblemIDs   []uint
}

// ProblemRef is the id/code/name of a problem included in a type.
type ProblemRef struct {
	ID   uint
	Code string
	Name string
}

// ListTypesResponse holds the response for listing types.
type ListTypesResponse struct {
	Types []ProblemType
	Total int64
}

// ListTypesRequest holds parameters for listing types.
type ListTypesRequest struct {
	Search string
}

// GetTypeRequest holds parameters for getting a type.
type GetTypeRequest struct {
	TypeID uint
}

// CreateTypeRequest holds parameters for creating a type.
type CreateTypeRequest struct {
	Name       string
	FullName   string
	ProblemIDs *[]uint
}

// UpdateTypeRequest holds parameters for updating a type.
type UpdateTypeRequest struct {
	TypeID     uint
	Name       *string
	FullName   *string
	ProblemIDs *[]uint
}

// DeleteTypeRequest holds parameters for deleting a type.
type DeleteTypeRequest struct {
	TypeID uint
}