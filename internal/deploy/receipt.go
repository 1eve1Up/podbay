package deploy

import (
	"strings"

	"github.com/1eve1Up/podbay/internal/expand"
	"github.com/1eve1Up/podbay/internal/receipt"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

// inspectContainerForReceipt is swappable in tests (default: podman inspect).
var inspectContainerForReceipt = runtimestate.InspectContainerForReceipt

func buildReceiptServiceRecords(r *runner.Runner, order []string, active map[string]spec.Service, host map[string]string) ([]receipt.ServiceRecord, error) {
	var out []receipt.ServiceRecord
	for _, name := range order {
		svc := expand.ExpandService(active[name], host)
		cname := r.ContainerName(name)
		cid, imgPodman, env, mounts, err := inspectContainerForReceipt(cname)
		if err != nil {
			return nil, err
		}
		img := strings.TrimSpace(svc.Image)
		if img == "" {
			img = imgPodman
		}
		out = append(out, receipt.ServiceRecord{
			Service:       name,
			ContainerName: cname,
			ContainerID:   cid,
			Image:         img,
			Env:           env,
			Mounts:        mounts,
		})
	}
	return out, nil
}
