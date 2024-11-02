package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	//"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-exec/tfexec"
)

const (
	DEFAULT_TF_CLI_VERSION = "1.9.8"
	TF_EXEC_PATH           = "/home/app/.bin/terraform"
	CACHE_MOUNPOINT        = "/opt/tfc-cache"

	// Define where terraform config and dotenv file are written on the disk
	SHARED_VOLUME_MOUNT_PATH string = "/opt/shared-volume"
	TF_CONFIG_REL_DIR_PATH   string = "/tfconfig"
	DOTENV_REL_FILE_PATH     string = "/.env"
)

// TO
func mustRunTfInit(tf *tfexec.Terraform) {
	defer timeTrack(time.Now(), "terraform-init")
	err := tf.Init(context.Background())
	if err != nil {
		log.Fatalf("Error running tf init: %t\n", err)
	}
	return
}

func tfInit() {
	defer timeTrack(time.Now(), "tf-init")

	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command("terraform", "init", "-no-color")
	cmd.Dir = filepath.Join(dirname, SHARED_VOLUME_MOUNT_PATH+TF_CONFIG_REL_DIR_PATH)
	stdout, err := cmd.Output()

	if err != nil {
		log.Println("Error while applying terraform init: " + err.Error())
	}
	// Print the output
	log.Println("Ouput of tf init: " + string(stdout))
}

func main() {
	tfConfigPath := SHARED_VOLUME_MOUNT_PATH + TF_CONFIG_REL_DIR_PATH

	// Install right version of tf cli
	tf, err := installTfCLI(tfConfigPath, true, DEFAULT_TF_CLI_VERSION)
	if err != nil {
		panic(err)
	}

	tfVersion, providerVersions, err := tf.Version(context.Background(), false)
	if err != nil {
		log.Fatalf("could not get terraform binary version info: %v", err)
	}
	fmt.Printf("Terraform CLI version is %v", tfVersion.String())
	_ = providerVersions

	// Load workspace variables
	if err := loadVariables(); err != nil {
	}
	fmt.Println("Workspace variables loaded successfully.")

	// terraform init
	// terraform plan
}

// Load environment variables from dotfile
func loadVariables() error {
	// TODO: load env vars from dotfile
	return nil
}

// path = Path to the directory that contains the Terraform configuration
func installTfCLI(path string, cache bool, defaultVer string) (*tfexec.Terraform, error) {
	defer timeTrack(time.Now(), "install-tf-cli")
	var cmd *exec.Cmd
	args := []string{"--default", defaultVer, "--chdir", path}
	if cache {
		args = append(args, "-install", CACHE_MOUNPOINT+"/terraform")
	}

	cmd = exec.Command("tfswitch", args...)
	if _, err := cmd.Output(); err != nil {
		return nil, fmt.Errorf("tfswitch could not install terraform cli, path=%v, cache=%t: %w", path, cache, err)
	}

	// Look for the 'terraform' binay in PATH
	tfExecPath, err := exec.LookPath("terraform")
	if err != nil {
		return nil, fmt.Errorf("could not find 'terraform' binary in PATH: %w", err)
	}

	// Create Terraform struct as provided by "hashicorp/terraform-exec"
	tf, err := tfexec.NewTerraform(tfExecPath, tfExecPath)
	if err != nil {
		return nil, fmt.Errorf("could not instantiate Terraform struct, path=%v: %w", path, err)
	}

	return tf, nil
}

func listDir(path string) {
	log.Printf("Listing content of dir=%v", path)
	entries, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, e := range entries {
		log.Println(e.Name())
	}
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %d ms", name, elapsed.Milliseconds())
}

func cleanConfig() {
	defer timeTrack(time.Now(), "cleanConfig")
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	err = os.RemoveAll(filepath.Join(dirname, SHARED_VOLUME_MOUNT_PATH+TF_CONFIG_REL_DIR_PATH))
	if err != nil {
		log.Println("Error while deleting all files of tf config: " + err.Error())
	}
}
