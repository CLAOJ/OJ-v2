// Package miscconfig provides miscellaneous configuration management services.
package miscconfig

// MiscConfig represents a key-value configuration entry.
type MiscConfig struct {
	ID    uint
	Key   string
	Value string
}

// ListConfigRequest holds parameters for listing configurations.
type ListConfigRequest struct {
	Page     int
	PageSize int
}

// ListConfigResponse holds the response for listing configurations.
type ListConfigResponse struct {
	Configs  []MiscConfig
	Total    int64
	Page     int
	PageSize int
}

// GetConfigRequest holds parameters for getting a configuration.
type GetConfigRequest struct {
	ConfigID uint
}

// GetConfigByKeyRequest holds parameters for getting a configuration by key.
type GetConfigByKeyRequest struct {
	Key string
}

// CreateConfigRequest holds parameters for creating a configuration.
type CreateConfigRequest struct {
	Key   string
	Value string
}

// UpdateConfigRequest holds parameters for updating a configuration.
type UpdateConfigRequest struct {
	ConfigID uint
	Value    string
}

// UpdateConfigByKeyRequest holds parameters for updating a configuration by key.
type UpdateConfigByKeyRequest struct {
	Key   string
	Value string
}

// DeleteConfigRequest holds parameters for deleting a configuration.
type DeleteConfigRequest struct {
	ConfigID uint
}

// DeleteConfigByKeyRequest holds parameters for deleting a configuration by key.
type DeleteConfigByKeyRequest struct {
	Key string
}
