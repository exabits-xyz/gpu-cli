package types

// VolumeType is a storage backend type returned by GET /volume-types.
type VolumeType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Volume represents a block-storage volume as returned by the Exabits API.
type Volume struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Bootable    bool   `json:"bootable"`
	Status      string `json:"status"` // creating, available, deleting, downloading, attaching, detaching, in-use
	SizeGB      int    `json:"size_gb"`
	Description string `json:"description,omitempty"`
	Region      string `json:"region"`
	CreatedTime int64  `json:"created_time"` // Unix timestamp
}

// VolumeListResult is the output shape for `volume list`.
type VolumeListResult struct {
	Total int      `json:"total"`
	Data  []Volume `json:"data"`
}

// CreateVolumeRequest is the body for POST /volumes.
type CreateVolumeRequest struct {
	DisplayName     string `json:"display_name"`
	RegionID        string `json:"region_id"`
	TypeID          string `json:"type_id"`
	Size            int    `json:"size"`
	ImageID         string `json:"image_id,omitempty"`
	Description     string `json:"description,omitempty"`
	PaymentCurrency string `json:"payment_currency,omitempty"`
}

// CreateVolumeResponse is the data object returned after a successful volume creation.
type CreateVolumeResponse struct {
	ID              string `json:"id"`
	DisplayName     string `json:"display_name"`
	Name            string `json:"name"`
	SizeGB          int    `json:"size_gb"`
	Status          string `json:"status"`
	Description     string `json:"description,omitempty"`
	PaymentCurrency string `json:"payment_currency,omitempty"`
}

// AttachVolumesRequest is the body for POST /virtual-machines/{id}/volumes.
type AttachVolumesRequest struct {
	VolumeIDs []string `json:"volume_ids"`
}

// DetachVolumeResponse is the data object returned after detaching a volume.
type DetachVolumeResponse struct {
	ID       string `json:"_id"`
	VolumeID string `json:"volume_id"`
	Status   string `json:"status"`
}
