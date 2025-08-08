package service

import (
	"fmt"
	"github.com/triapex/auth/model"
	"os"
	"os/exec"
)

func EnrollUser(namespace, org, ingressDomain, tlscertPath, tempDir string, userInfo *model.UserInfo) (string, error) {
	caAddress := fmt.Sprintf("%s-%s-ca-ca.%s", namespace, org, ingressDomain)
	enrollURL := fmt.Sprintf("https://%s:%s@%s", userInfo.Username, userInfo.Password, caAddress)

	cmd := exec.Command("fabric-ca-client", "enroll",
		"--url", enrollURL,
		"--tls.certfiles", tlscertPath,
		"--mspdir", tempDir,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		removeErr := os.RemoveAll(tempDir)
		if removeErr != nil {
			return "", fmt.Errorf("failed to remove temp dir: %v\nOutput: %s", removeErr, string(output))
		}
		return "", fmt.Errorf("enrollment failed: %v\nOutput: %s", err, string(output))
	}
	return tempDir, nil
}
func RegisterUser(namespace, org, ingressDomain, tlscertPath, rcamsPath string, userInfo *model.UserInfo) error {
	caAddress := fmt.Sprintf("%s-%s-ca-ca.%s", namespace, org, ingressDomain)

	cmd := exec.Command("fabric-ca-client", "register",
		"--id.name", userInfo.Username,
		"--id.secret", userInfo.Password,
		"--id.type", "client",
		"--id.affiliation", org,
		"--id.attrs", fmt.Sprintf("identity.id=%s:ecert", userInfo.ID),
		"--url", fmt.Sprintf("https://%s", caAddress),
		"--tls.certfiles", tlscertPath,
		"--mspdir", rcamsPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("registration failed: %v\nOutput: %s", err, string(output))
	}
	return nil
}
