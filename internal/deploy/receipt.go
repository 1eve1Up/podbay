package deploy

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/1eve1Up/podbay/internal/expand"
	"github.com/1eve1Up/podbay/internal/receipt"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

// inspectContainersForReceipt is swappable in tests (default: batched podman inspect).
var inspectContainersForReceipt = runtimestate.InspectContainersForReceipt

func buildReceiptServiceRecords(r *runner.Runner, order []string, active map[string]spec.Service, host map[string]string) ([]receipt.ServiceRecord, error) {
	names := make([]string, 0, len(order))
	for _, name := range order {
		names = append(names, r.ContainerName(name))
	}
	batch, err := inspectContainersForReceipt(names)
	if err != nil {
		return nil, err
	}
	var out []receipt.ServiceRecord
	for _, name := range order {
		svc := expand.ExpandService(active[name], host)
		cname := r.ContainerName(name)
		ri, ok := batch[cname]
		if !ok {
			return nil, fmt.Errorf("podman inspect %q: missing from batch", cname)
		}
		img := strings.TrimSpace(svc.Image)
		if img == "" {
			img = ri.Image
		}
		out = append(out, receipt.ServiceRecord{
			Service:       name,
			ContainerName: cname,
			ContainerID:   ri.ID,
			Image:         img,
			Env:           ri.Env,
			Mounts:        ri.Mounts,
		})
	}
	return out, nil
}

func writeDeployReceipt(ctx *deployContext, order []string) error {
	rp := strings.TrimSpace(ctx.opt.ReceiptPath)
	if rp == "" {
		return nil
	}
	deployID := receipt.NewDeployID()
	absReceipt, err := resolveReceiptWritePath(rp, deployID, time.Now().UTC())
	if err != nil {
		return err
	}
	absContract, err := filepath.Abs(ctx.contractFile)
	if err != nil {
		return fmt.Errorf("receipt: resolve contract path: %w", err)
	}
	records, err := buildReceiptServiceRecords(ctx.r, order, ctx.active, ctx.hostSubst)
	if err != nil {
		return fmt.Errorf("receipt: build records: %w", err)
	}
	digest, err := receipt.ContractDigestFile(absContract)
	if err != nil {
		return fmt.Errorf("receipt: %w", err)
	}
	rec := receipt.New(absContract, ctx.project, ctx.opt.Profiles, records)
	rec.DeployID = deployID
	rec.ContractDigest = digest
	rec.Status = receipt.StatusOK
	if ctx.partial {
		rec.DeployServices = append([]string(nil), ctx.opt.DeployServices...)
		rec.DependentsExpand = ctx.opt.DeployDependents
	}
	if err := receipt.WriteAtomic(absReceipt, rec); err != nil {
		return fmt.Errorf("receipt: %w", err)
	}
	if ctx.opt.WrittenReceiptPath != nil {
		*ctx.opt.WrittenReceiptPath = absReceipt
	}
	if !ctx.opt.Quiet {
		ctx.logf(" Wrote receipt %s", absReceipt)
	}
	return nil
}
