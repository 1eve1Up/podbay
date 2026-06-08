package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/1eve1Up/podbay/internal/receipt"
	"github.com/1eve1Up/podbay/internal/spec"
)

// contractPathOrArg resolves the contract path: -f/--file wins; otherwise first positional arg; else defaultFile.
func contractPathOrArg(fileFlag string, args []string, defaultFile string) (string, error) {
	if fileFlag != "" && len(args) > 0 {
		return "", fmt.Errorf("use either --file / -f or a path argument, not both")
	}
	if fileFlag != "" {
		return fileFlag, nil
	}
	if len(args) > 0 {
		return args[0], nil
	}
	return defaultFile, nil
}

// contractPathAndDeployServices resolves the contract file path and optional deploy/validate service roots.
// With --file / -f: every positional argument is a service name (zero means full stack).
// Without --file: [path] validates/deploys the full profile-active set; [path svc [svc...]] selects partial deploy roots.
func contractPathAndDeployServices(fileFlag string, args []string, defaultFile string) (path string, deployServices []string, err error) {
	if fileFlag != "" {
		return fileFlag, append([]string(nil), args...), nil
	}
	switch len(args) {
	case 0:
		return defaultFile, nil, nil
	case 1:
		arg := args[0]
		if contractLocationExists(arg) {
			return arg, nil, nil
		}
		// Disambiguate "service-name" vs "path-to-other-contract": if ./podbay.yaml
		// exists and parses, a matching service name takes the partial-deploy path.
		// A parse error on ./podbay.yaml is surfaced rather than silently falling
		// through to interpret the arg as a (likely nonexistent) contract path.
		if _, statErr := os.Stat(defaultFile); statErr == nil {
			c, _, loadErr := spec.Load(defaultFile)
			if loadErr != nil {
				return "", nil, augmentContractLoadError(defaultFile, loadErr)
			}
			if _, ok := c.Services[arg]; ok {
				return defaultFile, []string{arg}, nil
			}
		}
		return arg, nil, nil
	default:
		return args[0], append([]string(nil), args[1:]...), nil
	}
}

func loadContractWithDeployServices(fileFlag string, args []string, defaultFile string) (*spec.Contract, string, []string, error) {
	p, deployServices, err := contractPathAndDeployServices(fileFlag, args, defaultFile)
	if err != nil {
		return nil, "", nil, err
	}
	c, path, err := spec.Load(p)
	if err != nil {
		return nil, "", nil, augmentContractLoadError(p, err)
	}
	return c, path, deployServices, nil
}

// diffArgsDecodeAsReceiptPair reports whether args has two paths that both read and decode as deploy receipts.
func diffArgsDecodeAsReceiptPair(args []string) (pathA, pathB string, ok bool) {
	if len(args) != 2 {
		return "", "", false
	}
	data0, err0 := os.ReadFile(args[0])
	data1, err1 := os.ReadFile(args[1])
	if err0 != nil || err1 != nil {
		return "", "", false
	}
	if _, err := receipt.Decode(data0); err != nil {
		return "", "", false
	}
	if _, err := receipt.Decode(data1); err != nil {
		return "", "", false
	}
	return args[0], args[1], true
}

func loadContract(fileFlag string, args []string, defaultFile string) (*spec.Contract, string, error) {
	p, err := contractPathOrArg(fileFlag, args, defaultFile)
	if err != nil {
		return nil, "", err
	}
	c, path, err := spec.Load(p)
	if err != nil {
		return nil, "", augmentContractLoadError(p, err)
	}
	return c, path, nil
}

// expectedContractPath is the yaml file we would read for this user-supplied path (file, or dir + podbay.yaml).
func expectedContractPath(userPath string) string {
	abs, err := filepath.Abs(userPath)
	if err != nil {
		return userPath
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return abs
	}
	if fi.IsDir() {
		return filepath.Join(abs, spec.DefaultFilename)
	}
	return abs
}

// contractLocationExists reports whether userPath resolves to an existing contract file
// (a YAML file or a directory that contains podbay.yaml).
func contractLocationExists(userPath string) bool {
	p := expectedContractPath(userPath)
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func augmentContractLoadError(userPath string, loadErr error) error {
	if !errors.Is(loadErr, os.ErrNotExist) {
		return fmt.Errorf("load contract: %w", loadErr)
	}
	wd, _ := os.Getwd()
	tried := expectedContractPath(userPath)
	return fmt.Errorf("load contract: %w\n  cwd:           %s\n  expected file: %s\n  hint: -f is relative to cwd (above); cd to your app root or use e.g. -f myrepo/%s if the file lives in a subdirectory",
		loadErr, wd, tried, spec.DefaultFilename)
}

func projectName(c *spec.Contract, contractPath string) string {
	base := filepath.Base(filepath.Dir(contractPath))
	if base == "." || base == "" {
		wd, _ := os.Getwd()
		base = filepath.Base(wd)
	}
	return c.ProjectName(base)
}
