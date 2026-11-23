package monitor

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
)

func isExec(mode fs.FileMode) bool {
	return mode&0111 != 0
}

func findScontrol() (string, error) {

	path, err := exec.LookPath("scontrol")
	if err != nil {
		log.Printf("Couldn't find scontrol path checking default path")
	} else {
		return path, nil
	}
	fallbackPaths := []string{
		"/usr/local/slurm/latest/bin/scontrol",
		"/opt/slurm/bin/scontrol",
		"/usr/local/bin/scontrol",
		"/usr/bin/scontrol",
	}

	for _, fPath := range fallbackPaths {
		file, err := os.Stat(fPath)
		if err != nil {
			log.Printf("%s is not valid scontrol path, error : %v\n", fPath, err)
		} else if file.IsDir() || !isExec(file.Mode().Perm()) {
			log.Printf("%s is not a valid executable file\n", fPath)
		} else {
			return fPath, nil
		}
	}
	return "", fmt.Errorf("Scontrol is not present\n")

}
