package target

import (
	"github.com/prometheus/common/model"
)

// TargetType is the type of target
// nolint:revive
type TargetType string

const (
	// FileTargetType is a file target
	FileTargetType = TargetType("File")

	// JournalTargetType is a journalctl target
	JournalTargetType = TargetType("Journal")

	// SyslogTargetType is a syslog target
	SyslogTargetType = TargetType("Syslog")

	// GcplogTargetType is a target where log entries are pulled from pubsub topic.
	GcplogTargetType = TargetType("Gcplog")

	// DroppedTargetType is a target that's been dropped.
	DroppedTargetType = TargetType("dropped")

	// PushTargetType is a Loki push target
	PushTargetType = TargetType("Push")

	// WindowsTargetType is a Windows event target
	WindowsTargetType = TargetType("WindowsEvent")

	// KafkaTargetType is a Kafka target
	KafkaTargetType = TargetType("Kafka")

	// GelfTargetType is a gelf target
	GelfTargetType = TargetType("gelf")

	// CloudflareTargetType is a Cloudflare target
	CloudflareTargetType = TargetType("Cloudflare")

	// DockerTargetType is a Docker target
	DockerTargetType = TargetType("Docker")

	// HerokuDrainTargetType is a Heroku Logs target
	HerokuDrainTargetType = TargetType("HerokuDrain")

	HTTPTargetType = TargetType("HTTP")
)

// Target is a promtail scrape target
type Target interface {
	// Type of the target
	Type() TargetType
	// DiscoveredLabels returns labels discovered before any relabeling.
	DiscoveredLabels() model.LabelSet
	// Labels returns labels that are added to this target and its stream.
	Labels() model.LabelSet
	// Ready tells if the targets is ready
	Ready() bool
	// Details is additional information about this target specific to its type
	Details() interface{}
}

// IsDropped tells if a target has been dropped
func IsDropped(t Target) bool {
	return t.Type() == DroppedTargetType
}

// droppedTarget is a target that has been dropped
type droppedTarget struct {
	discoveredLabels model.LabelSet
	reason           string
}

func NewDroppedTarget(reason string, discoveredLabels model.LabelSet) Target {
	return &droppedTarget{
		discoveredLabels: discoveredLabels,
		reason:           reason,
	}
}

// Type implements Target
func (d *droppedTarget) Type() TargetType {
	return DroppedTargetType
}

// DiscoveredLabels implements Target
func (d *droppedTarget) DiscoveredLabels() model.LabelSet {
	return d.discoveredLabels
}

// Labels implements Target
func (d *droppedTarget) Labels() model.LabelSet {
	return nil
}

// Ready implements Target
func (d *droppedTarget) Ready() bool {
	return false
}

// Details implements Target it contains a message explaining the reason for dropping it
func (d *droppedTarget) Details() interface{} {
	return d.reason
}
