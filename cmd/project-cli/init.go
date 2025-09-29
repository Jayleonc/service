package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type manualStep struct {
	description string
	commands    []string
}

type gitRemote struct {
	name string
	url  string
	kind string
}

var templateRemoteIndicators = []string{
	"service-template",
	"project-template",
	"Jayleonc/service",
}

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

	var manualSteps []manualStep

	if dirName, err := projectDirectoryName(root); err == nil {
		renameQuestion := fmt.Sprintf("当前项目目录名为 %s，是否需要重命名？(y/N)", dirName)
		shouldRename, err := promptYesNo(reader, renameQuestion, false)
		if err != nil {
			return err
		}

		if shouldRename {
			newName, err := promptForDirectoryName(reader, dirName)
			if err != nil {
				return err
			}

			manualSteps = append(manualSteps, manualStep{
				description: "Rename the project directory:",
				commands:    []string{fmt.Sprintf("mv %s %s", dirName, newName)},
			})
		}
	} else {
		fmt.Printf("⚠️ Unable to determine project directory name: %v\n", err)
	}

	remotes, err := detectGitRemotes(root)
	if err != nil {
		fmt.Printf("⚠️ Unable to inspect Git remotes: %v\n", err)
	} else if len(remotes) == 0 {
		fmt.Println("未检测到任何 Git 远程仓库配置。")
	} else {
		for _, remote := range remotes {
			fmt.Printf("- %s (%s) -> %s\n", remote.name, remote.kind, remote.url)
		}

		for _, remote := range remotes {
			if looksLikeTemplateRemote(remote.url) {
				fmt.Printf("检测到 Git 远程 %q 指向模板仓库：%s\n", remote.name, remote.url)
				newRemote, err := promptForGitRemote(reader)
				if err != nil {
					return err
				}

				if newRemote != "" {
					manualSteps = append(manualSteps, manualStep{
						description: fmt.Sprintf("Update your Git remote %q:", remote.name),
						commands: []string{
							fmt.Sprintf("git remote remove %s", remote.name),
							fmt.Sprintf("git remote add %s %s", remote.name, newRemote),
						},
					})
				}
			}
		}
	}

	fmt.Println()
	fmt.Println("Summary:")
	if updatedFiles > 0 {
		fmt.Printf("- ✅ Go module path updated in %d files.\n", updatedFiles)
	} else {
		fmt.Println("- ℹ️ Go module path unchanged.")
	}

	if len(manualSteps) > 0 {
		fmt.Println()
		fmt.Println("Please perform the following final steps manually:")
		for i, step := range manualSteps {
			fmt.Printf("%d. %s\n", i+1, step.description)
			for _, cmd := range step.commands {
				fmt.Printf("   %s\n", cmd)
			}
		}
	} else {
		fmt.Println("No manual follow-up actions are required.")
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

func promptForDirectoryName(reader *bufio.Reader, current string) (string, error) {
	for {
		fmt.Print("请输入新的项目目录名: ")
		input, err := readLine(reader)
		if err != nil {
			return "", err
		}

		if input == "" {
			fmt.Println("Directory name cannot be empty.")
			continue
		}

		if input == current {
			fmt.Println("新目录名与当前目录名一致，如无需重命名请按 Enter 跳过。")
			continue
		}

		return input, nil
	}
}

func promptForGitRemote(reader *bufio.Reader) (string, error) {
	for {
		fmt.Print("请输入新的 Git 仓库地址 (留空跳过): ")
		input, err := readLine(reader)
		if err != nil {
			return "", err
		}

		if input == "" {
			return "", nil
		}

		return input, nil
	}
}

func looksLikeTemplateRemote(url string) bool {
	lowered := strings.ToLower(url)
	for _, indicator := range templateRemoteIndicators {
		if strings.Contains(lowered, strings.ToLower(indicator)) {
			return true
		}
	}
	return false
}

func detectGitRemotes(root string) ([]gitRemote, error) {
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = root

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git remote -v failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var remotes []gitRemote
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		remotes = append(remotes, gitRemote{
			name: fields[0],
			url:  fields[1],
			kind: strings.Trim(fields[2], "()"),
		})
	}

	return remotes, nil
}

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
