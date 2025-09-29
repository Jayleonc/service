package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// No longer needed manual steps or Git remote handling

func runInit(root string) error {
	fmt.Println("=== Project Initialization Wizard ===")

	modulePath, err := goModulePath(root)
	if err != nil {
		return fmt.Errorf("determine current module path: %w", err)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("当前 Go 模块路径: %s\n", modulePath)

	newPath, err := promptForModulePath(reader, modulePath)
	if err != nil {
		return err
	}

	var updatedFiles int
	if newPath != modulePath {
		fmt.Println("正在更新项目中的 Go 模块路径...")
		updatedFiles, err = replaceModulePath(root, modulePath, newPath)
		if err != nil {
			return err
		}
		fmt.Printf("✅ Go module path has been updated in %d files.\n", updatedFiles)
	} else {
		fmt.Println("Go module path unchanged; skipping file updates.")
	}

	// Delete .git folder to remove original repository configuration
	gitDir := filepath.Join(root, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		fmt.Println("正在删除原始仓库配置...")
		if err := os.RemoveAll(gitDir); err != nil {
			fmt.Printf("⚠️ 无法删除 .git 文件夹: %v\n", err)
		} else {
			fmt.Println("✅ 已删除原始仓库配置。您可以使用 'git init' 初始化新的仓库。")
		}
	} else if !os.IsNotExist(err) {
		fmt.Printf("⚠️ 检查 .git 文件夹时出错: %v\n", err)
	}

	// No manual steps needed anymore

	fmt.Println()
	fmt.Println("Summary:")
	if updatedFiles > 0 {
		fmt.Printf("- ✅ Go module path updated in %d files.\n", updatedFiles)
	} else {
		fmt.Println("- ℹ️ Go module path unchanged.")
	}

	// Add Git repository deletion to summary
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		fmt.Println("- ✅ 原始 Git 仓库配置已删除。")
	}

	fmt.Println("No manual follow-up actions are required.")

	fmt.Println()
	if dirName, err := projectDirectoryName(root); err == nil {
		fmt.Printf("💡 专业提示：建议在完成所有步骤后，手动将项目根目录 `%s` 重命名为具体的项目名称（例如: mv %s my_awesome_project）。\n", dirName, dirName)
	} else {
		fmt.Println("💡 专业提示：建议在完成所有步骤后，手动将项目根目录重命名为具体的项目名称（例如: mv service my_awesome_project）。")
	}

	fmt.Println()
	fmt.Println("Project initialisation complete. You can now comment out or remove the init-project target in the Makefile to prevent re-running it.")
	return nil
}

func promptForModulePath(reader *bufio.Reader, current string) (string, error) {
	for {
		fmt.Print("请输入新的Go模块路径 (例如: github.com/your-org/your-project): ")
		input, err := readLine(reader)
		if err != nil {
			return "", err
		}

		if input == "" {
			return current, nil
		}

		return input, nil
	}
}

// Git remote handling functions removed as they are no longer needed

func projectDirectoryName(root string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	return filepath.Base(abs), nil
}

func replaceModulePath(root, oldPath, newPath string) (int, error) {
	var updatedFiles int
	skipDirs := map[string]struct{}{
		".git":         {},
		"vendor":       {},
		"bin":          {},
		"tmp":          {},
		"node_modules": {},
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if path != root {
				if _, skip := skipDirs[d.Name()]; skip {
					return filepath.SkipDir
				}
			}
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if len(data) == 0 || isBinary(data) {
			return nil
		}

		if !bytes.Contains(data, []byte(oldPath)) {
			return nil
		}

		replaced := bytes.ReplaceAll(data, []byte(oldPath), []byte(newPath))
		if bytes.Equal(data, replaced) {
			return nil
		}

		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		if err := os.WriteFile(path, replaced, info.Mode()); err != nil {
			return err
		}

		updatedFiles++
		return nil
	})
	if err != nil {
		return 0, err
	}

	return updatedFiles, nil
}

func isBinary(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}

func goModulePath(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	return "", errors.New("module path not found in go.mod")
}
