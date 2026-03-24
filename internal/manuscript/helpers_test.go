package manuscript

import "os"

func mkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func chmodExec(path string) error {
	return os.Chmod(path, 0755)
}
