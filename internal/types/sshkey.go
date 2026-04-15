package types

// LocalSSHKey describes a key pair stored on disk under ~/.exabits/keys/.
type LocalSSHKey struct {
	Name           string `json:"name"`
	PrivateKeyPath string `json:"private_key_path"`
	PublicKeyPath  string `json:"public_key_path"`
	PublicKey      string `json:"public_key"`
	Fingerprint    string `json:"fingerprint"`
}
