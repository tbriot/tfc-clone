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
)

const (
	TF_EXEC_PATH    = "/home/app/.bin/terraform"
	CACHE_MOUNPOINT = "/opt/tfc-cache"

	// Define where terraform config and dotenv file are written on the disk
	SHARED_VOLUME_MOUNT_PATH string = "/opt/shared-volume"
	TF_CONFIG_REL_DIR_PATH   string = "/tfconfig"
	DOTENV_REL_FILE_PATH     string = "/.env"
)

func switchTfVersion(version string, cache bool) {
	defer timeTrack(time.Now(), "switchtfversion")
	var cmd *exec.Cmd
	if cache {
		cmd = exec.Command("tfswitch", "-i", CACHE_MOUNPOINT+"/terraform", version)
	} else {
		cmd = exec.Command("tfswitch", version)
	}
	_, err := cmd.Output()

	if err != nil {
		log.Println("tfswitch. error:" + err.Error())
	}
}

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
	// Install right version of tf cli
	version, err := installTfCLI()
	if err != nil {
	}
	fmt.Printf("Terraform CLI version=%v installed successfully.", version)

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

func installTfCLI() (version string, err error) {
	// Install proper tf CLI versions
	return "", nil
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
