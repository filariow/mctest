package testrun

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"

	cp "github.com/otiai10/copy"
)

func PrepareAndChdirIntoDotTmpTestsFolder(rootPath, tmpParentRootRelativePath string, copyOverFolders ...string) (*string, error) {
	d, err := PrepareDotTmpTestsFolder(rootPath, tmpParentRootRelativePath, copyOverFolders...)
	if err != nil {
		return nil, fmt.Errorf("error preparing .tmp folder at %s: %w", path.Join(rootPath, tmpParentRootRelativePath), err)
	}

	// move to .tmp folder
	if err := os.Chdir(*d); err != nil {
		return nil, fmt.Errorf("error changing directory to .tmp folder %s: %w", *d, err)
	}
	return d, nil
}

func PrepareDotTmpTestsFolder(rootPath, tmpParentRootRelativePath string, copyOverFolders ...string) (*string, error) {
	d := path.Join(rootPath, tmpParentRootRelativePath, ".tmp", "tests" /*, fmt.Sprintf("%d", time.Now().UTC().Unix())*/)
	wd, _ := os.Getwd()
	log.Printf("tests folder is %v", path.Join(wd, d))
	opts := cp.Options{
		AddPermission: 0666,
		OnDirExists: func(src, dest string) cp.DirExistsAction {
			return cp.Replace
		},
		PreserveOwner: true,
	}

	copyOverFunc := func(srcContext, destContext string, folder string) error {
		return cp.Copy(
			path.Join(srcContext, folder),
			path.Join(destContext, folder),
			opts)
	}

	removeDirFunc := func(path string) error {
		if err := os.RemoveAll(path); !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	baseFolder := path.Join(d, "base")
	preFolder := path.Join(d, "pre")

	// move .tmp/pre to .tmp/base
	if err := removeDirFunc(baseFolder); err != nil {
		return nil, err
	}
	if err := cp.Copy(preFolder, baseFolder, opts); err != nil {
		return nil, err
	}

	for _, f := range copyOverFolders {
		srcCtx := filepath.Dir(f)
		fn := filepath.Base(f)
		if err := copyOverFunc(path.Join(rootPath, srcCtx), baseFolder, fn); err != nil {
			return nil, err
		}
	}

	return &d, nil
}

func RemoveDotTmpFolder(basePath ...string) error {
	return os.RemoveAll(path.Join(path.Join(basePath...), ".tmp"))
}
