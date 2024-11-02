package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-exec/tfexec"
)

const (
	CACHE_MOUNPOINT   = "/opt/tfc-cache"
	TF_EXEC_PATH      = "/home/app/.bin/terraform"
	TF_CONFIG_DIRNAME = "/tf-config"
	VARIABLES_TABLE   = "vars"
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

func mustRunTfInit(tf *tfexec.Terraform) {
	defer timeTrack(time.Now(), "terraform-init")
	err := tf.Init(context.Background())
	if err != nil {
		log.Fatalf("Error running tf init: %t\n", err)
	}
	return
}

func cleanConfig() {
	defer timeTrack(time.Now(), "cleanConfig")
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	err = os.RemoveAll(filepath.Join(dirname, TF_CONFIG_DIRNAME))
	if err != nil {
		log.Println("Error while deleting all files of tf config: " + err.Error())
	}
}

func mustGetTFConfigDir() string {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(dirname, TF_CONFIG_DIRNAME)
}

func tfInit() {
	defer timeTrack(time.Now(), "tf-init")

	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command("terraform", "init", "-no-color")
	cmd.Dir = filepath.Join(dirname, TF_CONFIG_DIRNAME)
	stdout, err := cmd.Output()

	if err != nil {
		log.Println("Error while applying terraform init: " + err.Error())
	}
	// Print the output
	log.Println("Ouput of tf init: " + string(stdout))
}

func unzipTfConfigPackage(filepath string) {
	defer timeTrack(time.Now(), "unzip-tf-config")

	// create target directory if not existing
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	tfConfigDir := dirname + TF_CONFIG_DIRNAME
	_ = os.Mkdir(tfConfigDir, 0755)

	cmd := exec.Command("tar", "-xf", filepath, "--strip-components=1", "-C", tfConfigDir)
	_, err = cmd.Output()

	if err != nil {
		log.Println("Error while unzipping tf config package: " + err.Error())
	}
}

func main() {
	//listDir(CACHE_MOUNPOINT + "/terraform/.terraform.versions")

	for {
	}
}
