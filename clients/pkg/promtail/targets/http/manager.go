package http

import (
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/loki/v3/clients/pkg/logentry/stages"
	"github.com/grafana/loki/v3/clients/pkg/promtail/api"
	"github.com/grafana/loki/v3/clients/pkg/promtail/scrapeconfig"
	"github.com/grafana/loki/v3/clients/pkg/promtail/targets/target"
)

// TargetManager manages HTTP targets for scraping logs.
type TargetManager struct {
	logger  log.Logger
	metrics *Metrics
	targets map[string]*Target
}

// NewTargetManager creates a new HTTP target manager.
func NewTargetManager(
	metrics *Metrics,
	reg prometheus.Registerer,
	logger log.Logger,
	client api.EntryHandler,
	scrapeConfigs []scrapeconfig.Config,
) (*TargetManager, error) {
	tm := &TargetManager{
		logger:  logger,
		metrics: metrics,
		targets: make(map[string]*Target),
	}

	for _, cfg := range scrapeConfigs {
		pipeline, err := stages.NewPipeline(log.With(logger, "component", "http_pipeline_"+cfg.JobName), cfg.PipelineStages, &cfg.JobName, reg)
		if err != nil {
			return nil, err
		}

		t, err := NewTarget(metrics, logger, pipeline.Wrap(client), cfg.JobName, cfg.HTTPTargetConfig, cfg.RelabelConfigs)
		if err != nil {
			return nil, err
		}

		tm.targets[cfg.JobName] = t
	}

	return tm, nil
}

func (tm *TargetManager) Ready() bool {
	for _, t := range tm.targets {
		if t.Ready() {
			return true
		}
	}
	return false
}

func (tm *TargetManager) Stop() {
	for name, t := range tm.targets {
		if err := t.Stop(); err != nil {
			level.Error(t.logger).Log("event", "failed to stop http target", "name", name, "cause", err)
		}
	}
}

func (tm *TargetManager) ActiveTargets() map[string][]target.Target {
	return tm.AllTargets()
}

func (tm *TargetManager) AllTargets() map[string][]target.Target {
	res := make(map[string][]target.Target, len(tm.targets))
	for k, v := range tm.targets {
		res[k] = []target.Target{v}
	}
	return res
}
