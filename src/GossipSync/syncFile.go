package main

import (
	"fmt"
	"os/exec"
	"regexp"
)

func GetRelativePath(path string) string {
	compileRegex := regexp.MustCompile(`/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/\d+/fs(.*)`)
	matchArr := compileRegex.FindStringSubmatch(path)
	if len(matchArr) == 2 {
		return matchArr[1]
	} else {
		return ""
	}
}

func RunCmd(cmd *exec.Cmd) {
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("Error:%s\n输出: %s", err.Error(), output)
	} else {
		fmt.Println("指令输出：", output)
	}
}

func ModifiedFile(localHost string, localDir string, remoteHost string, remotePath string) {
	file := GetRelativePath(remotePath)
	localPath := localDir + file

	var cmd *exec.Cmd
	if localHost == remoteHost {
		cmd = exec.Command("cp", "-af", remotePath, localPath)
		fmt.Printf("更新：%s -> %s\n", remotePath, localPath)
	} else {
		remoteFullPath := fmt.Sprintf("root@%s:%s", remoteHost, remotePath)
		cmd = exec.Command("scp", "-p", remoteFullPath, localPath)
		fmt.Printf("更新：%s -> %s\n", remoteFullPath, localPath)
	}

	RunCmd(cmd)
}

func DeleteFile(localDir string, remotePath string) {
	file := GetRelativePath(remotePath)
	localPath := localDir + file

	cmd := exec.Command("rm", "-rf", localPath)
	fmt.Printf("删除：%s\n", localPath)
	RunCmd(cmd)
}
