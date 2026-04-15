package types

// VMSSHKey is the SSH key embedded in a VM's login block.
type VMSSHKey struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// VMLogin holds connection credentials returned with each VM.
type VMLogin struct {
	ID       string   `json:"_id"`
	SSHKey   VMSSHKey `json:"ssh_key"`
	Password string   `json:"password"`
	Username string   `json:"username"`
}

// VMFlavor describes the hardware spec of a VM.
type VMFlavor struct {
	Name     string `json:"name"`
	CPU      int    `json:"cpu"`
	RAM      int    `json:"ram"`  // GB
	Disk     int    `json:"disk"` // GB
	GPU      string `json:"gpu"`
	GPUCount int    `json:"gpu_count"`
}

// VMImage is the OS image a VM was launched from.
type VMImage struct {
	Name string `json:"name"`
}

// VMRegion is the datacenter region a VM is hosted in.
type VMRegion struct {
	Name string `json:"name"`
}

// VM represents a virtual machine instance as returned by the Exabits API.
type VM struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Login       VMLogin  `json:"login"`
	FixedIP     string   `json:"fixed_ip"`
	StartedTime int64    `json:"started_time"` // Unix timestamp
	Flavor      VMFlavor `json:"flavor"`
	Image       VMImage  `json:"image"`
	Region      VMRegion `json:"region"`
}

// VMListResult is the output shape for `vm list`.
// Total reflects the server-side record count before limit/offset are applied,
// so it may exceed len(Data) when paginating.
type VMListResult struct {
	Total int  `json:"total"`
	Data  []VM `json:"data"`
}

// SSHKeyInput is the ssh_key object sent in a CreateVMRequest.
// Distinct from VMSSHKey (read-side) which carries an ID instead of a public key.
type SSHKeyInput struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

// CreateVMRequest is the request body for POST /virtual-machines.
type CreateVMRequest struct {
	Name       string      `json:"name"`
	ImageID    string      `json:"image_id"`
	FlavorID   string      `json:"flavor_id"`
	SSHKey     SSHKeyInput `json:"ssh_key"`
	InitScript string      `json:"init_script,omitempty"`
}

// CreateVMResponse is the data object returned after a successful VM creation.
// The full VM detail is not returned at creation time; use `vm list` to inspect it.
type CreateVMResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LoginRequest is the body sent to POST /authenticate/login.
// Password must be pre-hashed with MD5 before being placed here.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"` // md5(plain-text password)
}

// LoginData is the data object returned inside a successful login response.
type LoginData struct {
	Username     string `json:"username"`
	Email        string `json:"email"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
