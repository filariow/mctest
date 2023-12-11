package testrun

import (
	"fmt"
	"os"
	"path"

	cp "github.com/otiai10/copy"
)

func PrepareAndChdirIntoDotTmpTestsFolder(basePath string) (*string, error) {
	d, err := PrepareDotTmpTestsFolder(basePath)
	if err != nil {
		return nil, fmt.Errorf("error preparing .tmp folder at %s: %w", basePath, err)
	}

	// move to .tmp folder
	if err := os.Chdir(*d); err != nil {
		return nil, fmt.Errorf("error changing directory to .tmp folder %s: %w", *d, err)
	}
	return d, nil
}

func PrepareDotTmpTestsFolder(basePath string) (*string, error) {
	d := path.Join(basePath, ".tmp", "tests" /*, fmt.Sprintf("%d", time.Now().UTC().Unix())*/)

	opts := cp.Options{
		AddPermission: 0600,
		OnDirExists: func(src, dest string) cp.DirExistsAction {
			return cp.Replace
		},
		PreserveOwner: true,
	}

	copyOverFunc := func(srcContext, destContext string, folder ...string) func() error {
		f := path.Join(folder...)
		return func() error {
			return cp.Copy(
				path.Join(srcContext, f),
				path.Join(destContext, f),
				opts)
		}
	}

	removeDirFunc := func(path string) func() error {
		return func() error {
			if err := os.RemoveAll(path); !os.IsNotExist(err) {
				return err
			}
			return nil
		}
	}

	baseFolder := path.Join(d, "base")
	preFolder := path.Join(d, "pre")
	todos := []func() error{
		removeDirFunc(baseFolder),
		func() error {
			if err := os.MkdirAll(path.Join(preFolder, "base"), 0755); !os.IsExist(err) {
				return err
			}
			return nil
		},
		copyOverFunc(path.Join(d, "pre"), d, "base"),
		removeDirFunc(preFolder),
		copyOverFunc("..", baseFolder, "member"),
		copyOverFunc("..", baseFolder, "host", "core"),
	}
	for _, f := range todos {
		if err := f(); err != nil {
			return nil, err
		}
	}

	return &d, nil
}

func RemoveDotTmpFolder(basePath ...string) error {
	return os.RemoveAll(path.Join(path.Join(basePath...), ".tmp"))
}
