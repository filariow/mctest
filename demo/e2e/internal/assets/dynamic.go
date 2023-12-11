package assets

import (
	"context"
	"os"
	"path"

	"github.com/filariow/mctest/pkg/testrun"
)

func ReadMemberCRDsFromConfigFolder(ctx context.Context) ([]byte, error) {
	f := testrun.TestFolderFromContextOrDie(ctx)
	return os.ReadFile(path.Join(f, "config", "crd", "member", "member.yaml"))
}
