package http

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/icholy/digest"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"

	"github.com/grafana/loki/v3/clients/pkg/promtail/api"
	"github.com/grafana/loki/v3/clients/pkg/promtail/scrapeconfig"
	"github.com/grafana/loki/v3/clients/pkg/promtail/targets/target"
	"github.com/grafana/loki/v3/pkg/logproto"
)

// Target scrapes logs from an HTTP endpoint using digest authentication.
type Target struct {
	logger         log.Logger
	handler        api.EntryHandler
	config         *scrapeconfig.HTTPTargetConfig
	jobName        string
	client         *http.Client
	metrics        *Metrics
	relabelConfigs []*relabel.Config
	ctx            context.Context // 添加上下文字段
}

// NewTarget creates a new HTTP target for scraping logs.
func NewTarget(metrics *Metrics, logger log.Logger, handler api.EntryHandler, jobName string, config *scrapeconfig.HTTPTargetConfig, relabel []*relabel.Config) (*Target, error) {
	wrappedLogger := log.With(logger, "component", "http_target", "job", jobName)

	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &digest.Transport{
			Username: config.Username,
			Password: config.Password,
		},
	}

	// 使用 context.Background() 或传入特定上下文
	ctx := context.Background()

	ht := &Target{
		metrics:        metrics,
		logger:         wrappedLogger,
		handler:        handler,
		jobName:        jobName,
		config:         config,
		client:         client,
		relabelConfigs: relabel,
		ctx:            ctx,
	}

	go ht.run()
	return ht, nil
}

func (h *Target) run() {
	level.Info(h.logger).Log("msg", "starting http target", "url", h.config.URL)

	ticker := time.NewTicker(h.config.PollInterval)
	defer ticker.Stop()

	// 使用 h.ctx 作为上下文
	ctx := h.ctx
	for {
		select {
		case <-ctx.Done():
			level.Info(h.logger).Log("msg", "http target stopped")
			return
		case <-ticker.C:
			if err := h.scrape(ctx); err != nil {
				level.Error(h.logger).Log("msg", "failed to scrape logs", "url", h.config.URL, "err", err)
				h.metrics.scrapeErrors.WithLabelValues(h.config.URL).Inc()
			}
		}
	}
}

func (h *Target) scrape(ctx context.Context) error {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 5 * time.Minute

	return backoff.Retry(func() error {
		start := time.Now()
		req, err := http.NewRequestWithContext(ctx, "GET", h.config.URL, nil)
		if err != nil {
			h.metrics.scrapeErrors.WithLabelValues(h.config.URL).Inc()
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := h.client.Do(req)
		h.metrics.requestDuration.WithLabelValues(h.config.URL).Observe(time.Since(start).Seconds())
		if err != nil {
			h.metrics.requestCounter.WithLabelValues(h.config.URL, "error").Inc()
			return fmt.Errorf("failed to fetch logs: %w", err)
		}
		defer resp.Body.Close()

		status := fmt.Sprintf("%d", resp.StatusCode)
		h.metrics.requestCounter.WithLabelValues(h.config.URL, status).Inc()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		entries := h.handler.Chan()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			lb := labels.NewBuilder(nil)
			lb.Set("__http_target_url", h.config.URL)
			if _, exists := h.config.Labels["job"]; !exists {
				lb.Set("job", h.jobName)
			}

			// Apply relabeling
			processed, _ := relabel.Process(lb.Labels(), h.relabelConfigs...)
			filtered := h.Labels().Clone()
			for _, lbl := range processed {
				if strings.HasPrefix(lbl.Name, "__") {
					continue
				}
				filtered[model.LabelName(lbl.Name)] = model.LabelValue(lbl.Value)
			}

			select {
			case entries <- api.Entry{
				Labels: filtered,
				Entry: logproto.Entry{
					Timestamp: time.Now(),
					Line:      line,
				},
			}:
				h.metrics.scrapeEntries.WithLabelValues(h.config.URL).Inc()
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err := scanner.Err(); err != nil {
			h.metrics.scrapeErrors.WithLabelValues(h.config.URL).Inc()
			return fmt.Errorf("failed to read response body: %w", err)
		}
		return nil
	}, backoff.WithContext(bo, ctx))
}

func (h *Target) Type() target.TargetType {
	return target.HTTPTargetType
}

func (h *Target) DiscoveredLabels() model.LabelSet {
	return nil
}

func (h *Target) Labels() model.LabelSet {
	return h.config.Labels
}

func (h *Target) Ready() bool {
	return h.config.URL != ""
}

func (h *Target) Details() interface{} {
	return map[string]string{
		"url": h.config.URL,
	}
}

func (h *Target) Stop() error {
	level.Info(h.logger).Log("msg", "stopping http target", "job", h.jobName)
	h.handler.Stop()
	return nil
}
