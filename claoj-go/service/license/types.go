// Package license provides license management services.
package license

// License represents a license.
type License struct {
	ID      uint
	Key     string
	Link    string
	Name    string
	Display string
	Icon    string
	Text    string
}

// ListLicensesRequest holds parameters for listing licenses.
type ListLicensesRequest struct {
	Page     int
	PageSize int
}

// ListLicensesResponse holds the response for listing licenses.
type ListLicensesResponse struct {
	Licenses []License
	Total    int64
	Page     int
	PageSize int
}

// GetLicenseRequest holds parameters for getting a license.
type GetLicenseRequest struct {
	LicenseID uint
}

// CreateLicenseRequest holds parameters for creating a license.
type CreateLicenseRequest struct {
	Key     string
	Link    string
	Name    string
	Display string
	Icon    string
	Text    string
}

// UpdateLicenseRequest holds parameters for updating a license.
type UpdateLicenseRequest struct {
	LicenseID uint
	Link      *string
	Name      *string
	Display   *string
	Icon      *string
	Text      *string
}

// DeleteLicenseRequest holds parameters for deleting a license.
type DeleteLicenseRequest struct {
	LicenseID uint
}
