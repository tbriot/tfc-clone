package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/joho/godotenv"
)

const (
	// Caching
	CACHE_MOUNPOINT       string = "/opt/tfc-cache"
	CACHE_TF_INSTALLATION bool   = true
	CACHE_TF_PLUGINS      bool   = true

	// Terraform CLI
	TF_EXEC_PATH             string = "/home/app/.bin/terraform"
	TF_VERSIONS_DIR_REL_PATH string = "/terraform"
	DEFAULT_TF_CLI_VERSION   string = "1.9.8"
	// Terraform plugins
	//TF_PLUGIN_CACHE_DIR_REL_PATH string = "/.terraform.d/plugin-cache"

	// Define where terraform config and dotenv file are written on the disk
	SHARED_VOLUME_MOUNT_PATH  string = "/opt/shared-volume"
	TF_CONFIG_REL_DIR_PATH    string = "/tfconfig"
	DOTENV_REL_FILE_PATH      string = "/.env"
	TF_PLAN_REL_FILE_PATH     string = "/tfplan"
	TF_PLAN_LOG_REL_FILE_PATH string = "/tfplan.log"
)

// TODO: get output in logs
func mustRunTfInit(tf *tfexec.Terraform) {
	defer timeTrack(time.Now(), "terraform-init")

	fmt.Println("Running terraform init command...")
	err := tf.Init(context.Background())
	if err != nil {
		log.Fatalf("Error running tf init: %t\n", err)
	}
	return
}

func mustRunTfPlan(tf *tfexec.Terraform) {
	defer timeTrack(time.Now(), "terraform-plan")

	tfPlanFilePath := SHARED_VOLUME_MOUNT_PATH + TF_PLAN_REL_FILE_PATH
	tfPlanLogFilePath := SHARED_VOLUME_MOUNT_PATH + TF_PLAN_LOG_REL_FILE_PATH

	logFileWriter, err := os.OpenFile(tfPlanLogFilePath, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatalf("Could not open file %v: %t\n", logFileWriter, err)
	}
	defer logFileWriter.Close()

	fmt.Printf("Running terraform plan command. Plan will be saved to %v...\n", tfPlanFilePath)
	_, err = tf.PlanJSON(context.Background(), logFileWriter, tfexec.Out(tfPlanFilePath))
	if err != nil {
		log.Fatalf("Error running tf plan: %t\n", err)
	}
	return
}

func main() {
	tfConfigPath := SHARED_VOLUME_MOUNT_PATH + TF_CONFIG_REL_DIR_PATH

	// Install right version of tf cli
	tf, err := installTfCLI(tfConfigPath, CACHE_TF_INSTALLATION, DEFAULT_TF_CLI_VERSION)
	if err != nil {
		panic(err)
	}

	tfVersion, providerVersions, err := tf.Version(context.Background(), false)
	if err != nil {
		log.Fatalf("could not get terraform binary version info: %v", err)
	}
	fmt.Printf("Terraform CLI version is %v.\n", tfVersion.String())
	_ = providerVersions

	// Load workspace variables
	if err := loadVariables(); err != nil {
		log.Fatalf("could not load workspace variables: %v", err)
	}
	fmt.Println("Workspace variables loaded successfully.")

	// terraform init
	mustRunTfInit(tf)

	// terraform plan
	mustRunTfPlan(tf)
}

// Load environment variables from .env file
func loadVariables() error {
	dotenvFilepath := filepath.Join(SHARED_VOLUME_MOUNT_PATH, DOTENV_REL_FILE_PATH)
	err := godotenv.Load(dotenvFilepath)
	if err != nil {
		return fmt.Errorf("Error loading .env file, filepath=%v: %w", dotenvFilepath, err)
	}
	return nil
}

// tfWorkingDir = Path to the directory that contains the Terraform configuration
func installTfCLI(tfWorkingDir string, cache bool, defaultVer string) (*tfexec.Terraform, error) {
	defer timeTrack(time.Now(), "install-tf-cli")
	tfVersionDirPath := filepath.Join(CACHE_MOUNPOINT, TF_VERSIONS_DIR_REL_PATH)
	var cmd *exec.Cmd
	args := []string{"--default", defaultVer, "--chdir", tfWorkingDir}
	if cache {
		args = append(args, "--install", tfVersionDirPath)
	}

	cmd = exec.Command("tfswitch", args...)
	if _, err := cmd.Output(); err != nil {
		return nil, fmt.Errorf(
			"tfswitch could not install terraform cli, tfWorkingDir=%v, cache=%t, tfVersionsDirPath=%v, cmd='%v': %w",
			tfWorkingDir, cache, tfVersionDirPath, cmd, err)
	}

	// Look for the 'terraform' binay in PATH
	tfExecPath, err := exec.LookPath("terraform")
	if err != nil {
		return nil, fmt.Errorf("could not find 'terraform' binary in PATH: %w", err)
	}

	// Create Terraform struct as provided by "hashicorp/terraform-exec"
	tf, err := tfexec.NewTerraform(tfWorkingDir, tfExecPath)
	if err != nil {
		return nil, fmt.Errorf("could not instantiate Terraform struct, tfWorkingDir=%v: %w", tfWorkingDir, err)
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
