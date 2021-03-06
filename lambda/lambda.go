package lambda

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Function struct {
	Path     string
	Name     string
	Target   string
	S3Bucket string
	S3Key    string
}

func (f *Function) Setup() error {
	defer f.cleanup()

	if err := f.compile(); err != nil {
		return fmt.Errorf("Error during function compilation: %s", err)
	}

	if err := f.install(); err != nil {
		return fmt.Errorf("Error during function installation: %s", err)
	}

	return nil
}

func (f *Function) compile() error {

	var out, stderr bytes.Buffer
	requirementsFile := fmt.Sprintf("%s/requirements.txt", f.Path)

	if _, err := os.Stat(requirementsFile); os.IsNotExist(err) {
		log.Printf("No requirements file found for %s, skipping", f.Path)
		return nil
	}
	cmd := exec.Command("pip",
		"install",
		"-r",
		requirementsFile,
		"-t",
		f.Name,
	)

	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	log.Printf(out.String())
	return nil
}

func (f *Function) install() error {
	var zipManager Zipper = LambdaZipper{}
	err := zipManager.Zip(f.Path, f.Target)
	if err != nil {
		return err
	}

	var uploader Uploader = S3Uploader{}
	err = uploader.Upload(f)
	if err != nil {
		return err
	}

	return nil
}

func (f *Function) cleanup() {
	log.Printf("[DEBUG] Cleaning up %s", f.Target)
	err := os.Remove(f.Target)
	if err != nil {
		log.Fatalf("Could not remove %s: %s", f.Path, err)
	}

}

func NewFunction(attrs map[string]string) *Function {
	path, ok := attrs["path"]
	if !ok {
		path = attrs["name"]
	}

	target := strings.Join([]string{attrs["name"], ".zip"}, "")
	return &Function{
		Name:     attrs["name"],
		Path:     path,
		S3Bucket: attrs["s3bucket"],
		Target:   target,
		S3Key:    strings.Join([]string{attrs["s3KeyPrefix"], target}, "/"),
	}
}
