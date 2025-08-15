package utils

import "os"

func CheckPath(path string) (isFile, isDir, exists bool) {
	info, err := os.Stat(path)
	if err != nil {
		return false, false, false
	}

	exists = true
	isDir = info.IsDir()
	isFile = !isDir
	return
}
