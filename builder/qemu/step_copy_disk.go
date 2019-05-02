package qemu

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
)

// This step copies the virtual disk that will be used as the
// hard drive for the virtual machine.
type stepCopyDisk struct{}

func (s *stepCopyDisk) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)
	isoPath := state.Get("iso_path").(string)
	ui := state.Get("ui").(packer.Ui)
	path := filepath.Join(config.OutputDir, fmt.Sprintf("%s", config.VMName))
	name := config.VMName

	// Check if the provided image is in the same format as the desired output
	// If yes, don't convert the image, simply copy it to the output path
	// This is also needed because `qemu-img convert` is broken on mac os x
	// See https://github.com/hashicorp/packer/issues/6963
	//     https://bugs.launchpad.net/qemu/+bug/1776920
	if format, err := driver.GetImageFormat(isoPath); err == nil && format == config.Format {
		if err = copyFile(isoPath, path); err != nil {
			err := fmt.Errorf("Error copying source file: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		return multistep.ActionContinue
	}

	command := []string{
		"convert",
		"-O", config.Format,
		isoPath,
		path,
	}

	if !config.DiskImage || config.UseBackingFile {
		return multistep.ActionContinue
	}

	ui.Say("Copying hard drive...")
	if err := driver.QemuImg(command...); err != nil {
		err := fmt.Errorf("Error creating hard drive: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("disk_filename", name)

	return multistep.ActionContinue
}

func (s *stepCopyDisk) Cleanup(state multistep.StateBag) {}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}
