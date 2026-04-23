package cmd

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/exabits-xyz/gpu-cli/internal/types"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

const keyDirName = "keys" // ~/.exabits/keys/

var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage local SSH key pairs for VM access",
}

// keyDir returns the resolved path to ~/.exabits/keys and creates it if absent.
func keyDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".exabits", keyDirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("cannot create key directory %s: %w", dir, err)
	}
	return dir, nil
}

// keyPaths returns the private and public key file paths for a given name.
func keyPaths(dir, name string) (priv, pub string) {
	base := filepath.Join(dir, "egpu_"+name)
	return base, base + ".pub"
}

// ── key generate ──────────────────────────────────────────────────────────────

var keyGenName string

var keyGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate an ed25519 SSH key pair",
	Long: `Generates a new ed25519 SSH key pair and saves both files under
~/.exabits/keys/:

  Private key → ~/.exabits/keys/egpu_<name>      (chmod 600)
  Public key  → ~/.exabits/keys/egpu_<name>.pub

To use the key when creating a VM:

  egpu vm create --name myvm --image-id <id> --flavor-id <id> \
                 --ssh-key-name <name> \
                 --ssh-public-key "$(cat ~/.exabits/keys/egpu_<name>.pub)"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if keyGenName == "" {
			exitInvalidArgs(fmt.Errorf("--name is required"))
			return nil
		}

		dir, err := keyDir()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		privPath, pubPath := keyPaths(dir, keyGenName)

		// Refuse to overwrite an existing key pair.
		if _, err := os.Stat(privPath); err == nil {
			exitInvalidArgs(fmt.Errorf(
				"key %q already exists at %s — delete it first or choose a different --name",
				keyGenName, privPath,
			))
			return nil
		}

		// Generate ed25519 key pair.
		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			exitAPIError(fmt.Errorf("key generation failed: %w", err))
			return nil
		}

		// Encode private key in OpenSSH PEM format.
		// ed25519.PrivateKey implements crypto.Signer directly.
		privPEMBlock, err := ssh.MarshalPrivateKey(privKey, "")
		if err != nil {
			exitAPIError(fmt.Errorf("failed to encode private key: %w", err))
			return nil
		}
		privPEMBytes := pem.EncodeToMemory(privPEMBlock)

		// Encode public key in authorized_keys format.
		sshPub, err := ssh.NewPublicKey(pubKey)
		if err != nil {
			exitAPIError(fmt.Errorf("failed to encode public key: %w", err))
			return nil
		}
		authorizedKey := strings.TrimRight(string(ssh.MarshalAuthorizedKey(sshPub)), "\n")
		fingerprint := ssh.FingerprintSHA256(sshPub)

		// Write private key — owner read/write only.
		if err := os.WriteFile(privPath, privPEMBytes, 0600); err != nil {
			exitAPIError(fmt.Errorf("failed to write private key: %w", err))
			return nil
		}
		// Write public key.
		if err := os.WriteFile(pubPath, []byte(authorizedKey+"\n"), 0644); err != nil {
			_ = os.Remove(privPath) // clean up partial write
			exitAPIError(fmt.Errorf("failed to write public key: %w", err))
			return nil
		}

		printJSON(types.LocalSSHKey{
			Name:           keyGenName,
			PrivateKeyPath: privPath,
			PublicKeyPath:  pubPath,
			PublicKey:      authorizedKey,
			Fingerprint:    fingerprint,
		})
		return nil
	},
}

// ── key list ──────────────────────────────────────────────────────────────────

var keyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List SSH key pairs stored in ~/.exabits/keys/",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := keyDir()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			exitAPIError(fmt.Errorf("cannot read key directory: %w", err))
			return nil
		}

		var keys []types.LocalSSHKey

		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".pub") {
				continue
			}

			pubPath := filepath.Join(dir, e.Name())
			raw, err := os.ReadFile(pubPath)
			if err != nil {
				continue // skip unreadable files
			}

			pubKeyStr := strings.TrimSpace(string(raw))
			sshPub, _, _, _, err := ssh.ParseAuthorizedKey(raw)
			if err != nil {
				continue // skip malformed files
			}

			// Derive key name from filename: egpu_<name>.pub → <name>
			base := strings.TrimSuffix(e.Name(), ".pub")
			name := strings.TrimPrefix(base, "egpu_")

			privPath := filepath.Join(dir, base)

			keys = append(keys, types.LocalSSHKey{
				Name:           name,
				PrivateKeyPath: privPath,
				PublicKeyPath:  pubPath,
				PublicKey:      pubKeyStr,
				Fingerprint:    ssh.FingerprintSHA256(sshPub),
			})
		}

		if keys == nil {
			keys = []types.LocalSSHKey{} // always emit an array, never null
		}

		printJSON(keys)
		return nil
	},
}

// ── key delete ────────────────────────────────────────────────────────────────

var keyDeleteForce bool

var keyDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a local SSH key pair from ~/.exabits/keys/",
	Long: `Deletes both the private key and the public key for the given name:

  ~/.exabits/keys/egpu_<name>
  ~/.exabits/keys/egpu_<name>.pub

In an interactive terminal you will be prompted to confirm.
Pass --force to skip the prompt (required when stdin is not a TTY).`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			exitInvalidArgs(fmt.Errorf("requires exactly 1 argument: <name>"))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if !keyDeleteForce {
			// Interactive terminal: prompt for y/n confirmation.
			if stdinIsTTY() {
				fmt.Fprintf(os.Stderr, "Delete key pair %q from ~/.exabits/keys/? [y/N]: ", name)
				input, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				input = strings.ToLower(strings.TrimSpace(input))
				if input != "y" {
					printJSON(map[string]string{
						"name":    name,
						"status":  "cancelled",
						"message": "deletion cancelled",
					})
					return nil
				}
			} else {
				// Non-interactive (piped/agent): require --force to avoid blocking.
				exitInvalidArgs(fmt.Errorf(
					"stdin is not a TTY — pass --force to confirm deletion non-interactively",
				))
				return nil
			}
		}

		dir, err := keyDir()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		privPath, pubPath := keyPaths(dir, name)

		// At least one of the two files must exist.
		_, privErr := os.Stat(privPath)
		_, pubErr := os.Stat(pubPath)
		if os.IsNotExist(privErr) && os.IsNotExist(pubErr) {
			exitAPIError(fmt.Errorf("no key named %q found in %s", name, dir))
			return nil
		}

		var errs []string
		if err := os.Remove(privPath); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("private key: %v", err))
		}
		if err := os.Remove(pubPath); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("public key: %v", err))
		}
		if len(errs) > 0 {
			exitAPIError(fmt.Errorf("failed to delete: %s", strings.Join(errs, "; ")))
			return nil
		}

		printJSON(map[string]string{
			"name":    name,
			"status":  "deleted",
			"message": fmt.Sprintf("key pair %q deleted from %s", name, dir),
		})
		return nil
	},
}

// stdinIsTTY reports whether stdin is an interactive terminal.
// Returns false when stdin is piped or redirected (e.g. in agent/CI use).
func stdinIsTTY() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func init() {
	// key generate flags
	keyGenerateCmd.Flags().StringVar(&keyGenName, "name", "", "Key name — files are saved as egpu_<name> and egpu_<name>.pub (required)")

	// key delete flags
	keyDeleteCmd.Flags().BoolVar(&keyDeleteForce, "force", false, "Confirm deletion of both key files (required)")

	keyCmd.AddCommand(keyGenerateCmd)
	keyCmd.AddCommand(keyListCmd)
	keyCmd.AddCommand(keyDeleteCmd)

	rootCmd.AddCommand(keyCmd)
}
