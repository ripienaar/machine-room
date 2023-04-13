package facts

import (
	"context"
	"os"
	"path/filepath"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/tokens"
	"github.com/ripienaar/machine-room/pkg/options"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/sirupsen/logrus"
)

// TODO: support users disabling our default fact gathering entirely if they supply a generator

func Generate(ctx context.Context, cfg options.Options, log *logrus.Entry) (any, error) {
	data := map[string]map[string]any{
		"host":         {},
		"mem":          {},
		"swap":         {},
		"cpu":          {},
		"disk":         {},
		"net":          {},
		"machine_room": {},
	}
	var err error

	data["mem"]["virtual"], err = mem.VirtualMemory()
	if err != nil {
		log.Warnf("Could not gather virtual memory information: %v", err)
	}

	data["swap"]["memory"], err = mem.SwapMemory()
	if err != nil {
		log.Warnf("Could not gather swap information: %v", err)
	}

	data["cpu"]["info"], err = cpu.Info()
	if err != nil {
		log.Warnf("Could not gather CPU information: %v", err)
	}

	parts, err := disk.Partitions(true)
	if err != nil {
		log.Warnf("Could not gather Disk partitions: %v", err)
	}
	if len(parts) > 0 {
		matchedParts := []disk.PartitionStat{}
		usages := []*disk.UsageStat{}

		for _, part := range parts {
			if part.Mountpoint == "" || part.Fstype == "tmpfs" || part.Fstype == "cgroup" || part.Fstype == "proc" || part.Fstype == "devpts" || part.Fstype == "sysfs" || part.Fstype == "mqueue" {
				continue
			}
			matchedParts = append(matchedParts, part)
			u, err := disk.Usage(part.Mountpoint)
			if err != nil {
				log.Warnf("Could not get usage for partition %s: %v", part.Mountpoint, err)
				continue
			}
			usages = append(usages, u)
		}

		data["disk"]["partitions"] = matchedParts
		data["disk"]["usage"] = usages
	}

	data["host"]["info"], err = host.Info()
	if err != nil {
		log.Warnf("Could not gather host information: %v", err)
	}

	data["net"]["interfaces"], err = net.Interfaces()
	if err != nil {
		log.Warnf("Could not gather network interfaces: %v", err)
	}

	ext := tokens.MapClaims{}
	if choria.FileExist(cfg.ProvisioningJWTFile) {
		td, err := os.ReadFile(cfg.ProvisioningJWTFile)
		if err == nil {
			t, err := tokens.ParseProvisionTokenUnverified(string(td))
			if err == nil {
				ext = t.Extensions
			}
		}
	}

	token, err := os.ReadFile(filepath.Join(filepath.Dir(cfg.ProvisioningJWTFile), "server.jwt"))
	if err != nil {
		log.Warnf("Could not read server token: %v", err)
	}

	data["machine_room"] = map[string]any{
		"server_token": string(token),
		"provisioning": map[string]any{
			"extended_claims": ext,
		},
	}

	if cfg.AdditionalFacts != nil {
		extra, err := cfg.AdditionalFacts(ctx, cfg, log)
		if err != nil {
			log.Errorf("Could not gather additional facts: %v", err)
		} else {
			data["machine_room"]["additional_facts"] = extra
		}
	}

	return data, nil
}
