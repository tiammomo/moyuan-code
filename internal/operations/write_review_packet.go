package operations

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"moyuan-code/internal/evidence"
	"moyuan-code/internal/fsutil"
	"moyuan-code/internal/logging"
	"moyuan-code/internal/workspace"
)

type WriteReviewPacketOptions struct {
	AdmissionID   string `json:"admission_id,omitempty"`
	OperationType string `json:"operation_type,omitempty"`
	OperationID   string `json:"operation_id,omitempty"`
	Provider      string `json:"provider,omitempty"`
	Environment   string `json:"environment,omitempty"`
	Status        string `json:"status,omitempty"`
	Decision      string `json:"decision,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}

type WriteReviewPacketReport struct {
	ID          string                   `json:"id"`
	GeneratedAt string                   `json:"generated_at"`
	Filters     WriteReviewPacketOptions `json:"filters"`
	Summary     WriteReviewPacketSummary `json:"summary"`
	Packets     []WriteReviewPacket      `json:"packets"`
}

type WriteReviewPacketSummary struct {
	PacketCount         int            `json:"packet_count"`
	ReadyCount          int            `json:"ready_count"`
	BlockedCount        int            `json:"blocked_count"`
	ManualRequiredCount int            `json:"manual_required_count"`
	ByOperationType     map[string]int `json:"by_operation_type,omitempty"`
	ByProvider          map[string]int `json:"by_provider,omitempty"`
	ByEnvironment       map[string]int `json:"by_environment,omitempty"`
	ByStatus            map[string]int `json:"by_status,omitempty"`
	ByDecision          map[string]int `json:"by_decision,omitempty"`
}

type WriteReviewPacket struct {
	ID                         string                     `json:"id"`
	AdmissionID                string                     `json:"admission_id"`
	ProofID                    string                     `json:"proof_id,omitempty"`
	ProofDecision              string                     `json:"proof_decision,omitempty"`
	OperationType              string                     `json:"operation_type"`
	OperationID                string                     `json:"operation_id"`
	Provider                   string                     `json:"provider,omitempty"`
	Environment                string                     `json:"environment,omitempty"`
	Mode                       string                     `json:"mode,omitempty"`
	Status                     string                     `json:"status"`
	Decision                   string                     `json:"decision"`
	Reasons                    []string                   `json:"reasons,omitempty"`
	RuleRefs                   []string                   `json:"rule_refs,omitempty"`
	EvidenceRefs               []string                   `json:"evidence_refs,omitempty"`
	SourceRef                  string                     `json:"source_ref,omitempty"`
	WriteEnabled               bool                       `json:"write_enabled"`
	RehearsalAllowed           bool                       `json:"rehearsal_allowed"`
	ApprovalRequired           bool                       `json:"approval_required"`
	ApprovalSatisfied          bool                       `json:"approval_satisfied"`
	ApprovalID                 string                     `json:"approval_id,omitempty"`
	ProviderRequirementID      string                     `json:"provider_requirement_id,omitempty"`
	ProviderRequirementVersion string                     `json:"provider_requirement_version,omitempty"`
	ProviderRequirementRefs    []string                   `json:"provider_requirement_refs,omitempty"`
	RemoteRehearsalID          string                     `json:"remote_rehearsal_id,omitempty"`
	RemoteRehearsalStatus      string                     `json:"remote_rehearsal_status,omitempty"`
	RemoteRehearsalDecision    string                     `json:"remote_rehearsal_decision,omitempty"`
	QueueItems                 []WriteReviewQueueSnapshot `json:"queue_items,omitempty"`
	QueueItemIDs               []string                   `json:"queue_item_ids,omitempty"`
	QueueDecisions             []string                   `json:"queue_decisions,omitempty"`
	Markdown                   string                     `json:"markdown,omitempty"`
	CreatedAt                  string                     `json:"created_at"`
	Metadata                   map[string]any             `json:"metadata,omitempty"`
}

type WriteReviewQueueSnapshot struct {
	ID                    string   `json:"id"`
	Status                string   `json:"status"`
	Decision              string   `json:"decision"`
	Trigger               string   `json:"trigger,omitempty"`
	Environment           string   `json:"environment,omitempty"`
	DeploymentExecutionID string   `json:"deployment_execution_id,omitempty"`
	AdmissionID           string   `json:"admission_id,omitempty"`
	RemoteRehearsalID     string   `json:"remote_rehearsal_id,omitempty"`
	ReviewPacketID        string   `json:"review_packet_id,omitempty"`
	RunID                 string   `json:"run_id,omitempty"`
	Reasons               []string `json:"reasons,omitempty"`
	CreatedAt             string   `json:"created_at,omitempty"`
	UpdatedAt             string   `json:"updated_at,omitempty"`
}

func CreateWriteReviewPackets(rootDir string, options WriteReviewPacketOptions) (WriteReviewPacketReport, error) {
	options = normalizeWriteReviewPacketOptions(options)
	admissions, err := BuildWriteAdmissions(rootDir, WriteAdmissionOptions{
		Provider:      options.Provider,
		OperationType: options.OperationType,
		Environment:   options.Environment,
		Limit:         options.Limit,
	})
	if err != nil {
		return WriteReviewPacketReport{}, err
	}
	remoteReport, err := ListRemoteExecutionRehearsals(rootDir, RemoteExecutionRehearsalOptions{Limit: 100})
	if err != nil {
		return WriteReviewPacketReport{}, err
	}
	queues, err := listWriteReviewQueueSnapshots(rootDir)
	if err != nil {
		return WriteReviewPacketReport{}, err
	}

	packets := []WriteReviewPacket{}
	for _, admission := range admissions.Entries {
		if options.AdmissionID != "" && admission.ID != options.AdmissionID {
			continue
		}
		if options.OperationID != "" && admission.OperationID != options.OperationID {
			continue
		}
		packet := writeReviewPacketFromAdmission(admission, latestRemoteRehearsalForOperation(remoteReport.Rehearsals, admission.OperationID), queues)
		if !writeReviewPacketMatches(packet, options) {
			continue
		}
		packet, err = finishWriteReviewPacket(rootDir, packet)
		if err != nil {
			return WriteReviewPacketReport{}, err
		}
		packets = append(packets, packet)
	}
	sortWriteReviewPackets(packets)
	if len(packets) > options.Limit {
		packets = packets[:options.Limit]
	}
	now := time.Now().UTC()
	report := WriteReviewPacketReport{
		ID:          "write-review-packet-report-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		GeneratedAt: now.Format(time.RFC3339Nano),
		Filters:     options,
		Packets:     packets,
	}
	report.Summary = buildWriteReviewPacketSummary(packets)
	return report, nil
}

func ListWriteReviewPackets(rootDir string, options WriteReviewPacketOptions) (WriteReviewPacketReport, error) {
	options = normalizeWriteReviewPacketOptions(options)
	if err := fsutil.EnsureDir(writeReviewPacketDir(rootDir)); err != nil {
		return WriteReviewPacketReport{}, err
	}
	entries, err := os.ReadDir(writeReviewPacketDir(rootDir))
	if err != nil {
		return WriteReviewPacketReport{}, err
	}
	packets := []WriteReviewPacket{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var packet WriteReviewPacket
		found, err := fsutil.ReadJSON(filepath.Join(writeReviewPacketDir(rootDir), entry.Name()), &packet)
		if err != nil {
			return WriteReviewPacketReport{}, err
		}
		if found && packet.ID != "" && writeReviewPacketMatches(packet, options) {
			packets = append(packets, packet)
		}
	}
	sortWriteReviewPackets(packets)
	if len(packets) > options.Limit {
		packets = packets[:options.Limit]
	}
	now := time.Now().UTC()
	report := WriteReviewPacketReport{
		ID:          "write-review-packet-list-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		GeneratedAt: now.Format(time.RFC3339Nano),
		Filters:     options,
		Packets:     packets,
	}
	report.Summary = buildWriteReviewPacketSummary(packets)
	return report, nil
}

func LoadWriteReviewPacket(rootDir string, id string) (WriteReviewPacket, bool, error) {
	var packet WriteReviewPacket
	found, err := fsutil.ReadJSON(writeReviewPacketPath(rootDir, strings.TrimSpace(id)), &packet)
	return packet, found, err
}

func writeReviewPacketFromAdmission(admission WriteAdmissionEntry, rehearsal RemoteExecutionRehearsal, queues []WriteReviewQueueSnapshot) WriteReviewPacket {
	now := time.Now().UTC()
	packet := WriteReviewPacket{
		ID:                         "write-review-packet-" + admission.OperationType + "-" + admission.OperationID + "-" + now.Format("20060102150405") + "-" + fmt.Sprintf("%09d", now.UnixNano()%1_000_000_000),
		AdmissionID:                admission.ID,
		ProofID:                    admission.ProofID,
		ProofDecision:              admission.ProofDecision,
		OperationType:              admission.OperationType,
		OperationID:                admission.OperationID,
		Provider:                   admission.Provider,
		Environment:                admission.Environment,
		Mode:                       admission.Mode,
		Status:                     "ready",
		Decision:                   "WRITE_REVIEW_PACKET_READY",
		Reasons:                    []string{"write_review_packet_sources_collected"},
		RuleRefs:                   append([]string{}, admission.RuleRefs...),
		EvidenceRefs:               append([]string{}, admission.ProviderEvidenceRefs...),
		SourceRef:                  admission.SourceRef,
		WriteEnabled:               admission.WriteEnabled,
		RehearsalAllowed:           admission.RehearsalAllowed,
		ApprovalRequired:           admission.ApprovalRequired,
		ApprovalSatisfied:          admission.ApprovalSatisfied,
		ApprovalID:                 admission.ApprovalID,
		ProviderRequirementID:      admission.ProviderRequirementID,
		ProviderRequirementVersion: admission.ProviderRequirementVersion,
		ProviderRequirementRefs:    append([]string{}, admission.ProviderRequirementRefs...),
		QueueItems:                 []WriteReviewQueueSnapshot{},
		QueueItemIDs:               []string{},
		QueueDecisions:             []string{},
		CreatedAt:                  now.Format(time.RFC3339Nano),
		Metadata: map[string]any{
			"source_admission_status":   admission.Status,
			"source_admission_decision": admission.Decision,
			"dry_run":                   admission.DryRun,
			"secret_ref_status":         admission.SecretRefStatus,
		},
	}
	blockPacket := func(decision string, reason string, rule string) {
		packet.Status = "blocked"
		packet.Decision = decision
		packet.Reasons = appendUnique(packet.Reasons, reason)
		packet.RuleRefs = appendUnique(packet.RuleRefs, rule)
	}
	manualPacket := func(decision string, reason string, rule string) {
		if packet.Status != "blocked" {
			packet.Status = "manual_required"
			packet.Decision = decision
		}
		packet.Reasons = appendUnique(packet.Reasons, reason)
		packet.RuleRefs = appendUnique(packet.RuleRefs, rule)
	}

	switch admission.Status {
	case "blocked":
		blockPacket("WRITE_REVIEW_PACKET_ADMISSION_BLOCKED", "source_write_admission_blocked:"+admission.Decision, "source_write_admission_must_be_ready")
	case "manual_required":
		manualPacket("WRITE_REVIEW_PACKET_ADMISSION_MANUAL_REQUIRED", "source_write_admission_manual_required:"+admission.Decision, "source_write_admission_manual_review")
	case "rehearsal_only":
		manualPacket("WRITE_REVIEW_PACKET_REHEARSAL_ONLY", "source_write_admission_rehearsal_only:"+admission.Decision, "real_write_requires_ready_admission")
	}

	if admission.OperationType == "deployment_execution" {
		if rehearsal.ID == "" {
			manualPacket("WRITE_REVIEW_PACKET_REMOTE_REHEARSAL_REQUIRED", "remote_execution_rehearsal_missing", "remote_rehearsal_required_before_write")
		} else {
			packet.RemoteRehearsalID = rehearsal.ID
			packet.RemoteRehearsalStatus = rehearsal.Status
			packet.RemoteRehearsalDecision = rehearsal.Decision
			packet.EvidenceRefs = appendUniqueStrings(packet.EvidenceRefs, rehearsal.EvidenceRefs...)
			packet.RuleRefs = appendUniqueStrings(packet.RuleRefs, rehearsal.RuleRefs...)
			if rehearsal.Status == "blocked" {
				blockPacket("WRITE_REVIEW_PACKET_REMOTE_REHEARSAL_BLOCKED", "remote_execution_rehearsal_blocked:"+rehearsal.Decision, "remote_rehearsal_must_pass")
			}
			if rehearsal.Status == "manual_required" {
				manualPacket("WRITE_REVIEW_PACKET_REMOTE_REHEARSAL_MANUAL_REQUIRED", "remote_execution_rehearsal_manual_required:"+rehearsal.Decision, "remote_rehearsal_manual_review")
			}
			if rehearsal.Status == "completed" {
				packet.Reasons = appendUnique(packet.Reasons, "remote_execution_rehearsal_completed")
			}
		}
	}

	for _, queue := range queues {
		if !writeReviewQueueMatchesAdmission(queue, admission) {
			continue
		}
		packet.QueueItems = append(packet.QueueItems, queue)
		packet.QueueItemIDs = appendUnique(packet.QueueItemIDs, queue.ID)
		packet.QueueDecisions = appendUnique(packet.QueueDecisions, queue.Decision)
		switch queue.Status {
		case "manual_required":
			manualPacket("WRITE_REVIEW_PACKET_QUEUE_MANUAL_REQUIRED", "control_queue_manual_required:"+queue.Decision, "control_queue_must_be_executable")
		case "waiting":
			packet.Reasons = appendUnique(packet.Reasons, "control_queue_waiting:"+queue.Decision)
		case "completed":
			packet.Reasons = appendUnique(packet.Reasons, "control_queue_completed:"+queue.Decision)
		}
	}

	if packet.Status == "ready" {
		packet.Reasons = appendUnique(packet.Reasons, "write_review_packet_ready")
		packet.RuleRefs = appendUnique(packet.RuleRefs, "all_review_packet_gates_satisfied")
	}
	packet.Markdown = renderWriteReviewPacketMarkdown(packet)
	return normalizeWriteReviewPacket(packet)
}

func normalizeWriteReviewPacketOptions(options WriteReviewPacketOptions) WriteReviewPacketOptions {
	options.AdmissionID = strings.TrimSpace(options.AdmissionID)
	options.OperationType = normalizeType(options.OperationType)
	options.OperationID = strings.TrimSpace(options.OperationID)
	options.Provider = normalizeType(options.Provider)
	options.Environment = normalizeType(options.Environment)
	options.Status = normalizeType(options.Status)
	options.Decision = normalizeType(options.Decision)
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Limit > 100 {
		options.Limit = 100
	}
	return options
}

func normalizeWriteReviewPacket(packet WriteReviewPacket) WriteReviewPacket {
	packet.Provider = normalizeType(packet.Provider)
	packet.OperationType = normalizeType(packet.OperationType)
	packet.Environment = normalizeType(packet.Environment)
	packet.Mode = normalizeType(packet.Mode)
	packet.Status = normalizeType(packet.Status)
	packet.EvidenceRefs = compactStrings(packet.EvidenceRefs)
	packet.RuleRefs = compactStrings(packet.RuleRefs)
	packet.ProviderRequirementRefs = compactStrings(packet.ProviderRequirementRefs)
	packet.QueueItemIDs = compactStrings(packet.QueueItemIDs)
	packet.QueueDecisions = compactStrings(packet.QueueDecisions)
	return packet
}

func writeReviewPacketMatches(packet WriteReviewPacket, options WriteReviewPacketOptions) bool {
	if options.AdmissionID != "" && packet.AdmissionID != options.AdmissionID {
		return false
	}
	if options.OperationType != "" && normalizeType(packet.OperationType) != options.OperationType {
		return false
	}
	if options.OperationID != "" && packet.OperationID != options.OperationID {
		return false
	}
	if options.Provider != "" && normalizeType(packet.Provider) != options.Provider {
		return false
	}
	if options.Environment != "" && normalizeType(packet.Environment) != options.Environment {
		return false
	}
	if options.Status != "" && normalizeType(packet.Status) != options.Status {
		return false
	}
	if options.Decision != "" && normalizeType(packet.Decision) != options.Decision {
		return false
	}
	return true
}

func buildWriteReviewPacketSummary(packets []WriteReviewPacket) WriteReviewPacketSummary {
	summary := WriteReviewPacketSummary{
		PacketCount:     len(packets),
		ByOperationType: map[string]int{},
		ByProvider:      map[string]int{},
		ByEnvironment:   map[string]int{},
		ByStatus:        map[string]int{},
		ByDecision:      map[string]int{},
	}
	for _, packet := range packets {
		summary.ByOperationType[packet.OperationType]++
		if packet.Provider != "" {
			summary.ByProvider[packet.Provider]++
		}
		if packet.Environment != "" {
			summary.ByEnvironment[packet.Environment]++
		}
		summary.ByStatus[packet.Status]++
		summary.ByDecision[packet.Decision]++
		switch packet.Status {
		case "ready":
			summary.ReadyCount++
		case "blocked":
			summary.BlockedCount++
		case "manual_required":
			summary.ManualRequiredCount++
		}
	}
	return summary
}

func latestRemoteRehearsalForOperation(rehearsals []RemoteExecutionRehearsal, operationID string) RemoteExecutionRehearsal {
	for _, rehearsal := range rehearsals {
		if rehearsal.OperationID == operationID {
			return rehearsal
		}
	}
	return RemoteExecutionRehearsal{}
}

func listWriteReviewQueueSnapshots(rootDir string) ([]WriteReviewQueueSnapshot, error) {
	dir := filepath.Join(workspace.ForRoot(rootDir).ControlLoopDir, "queue")
	if err := fsutil.EnsureDir(dir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	queues := []WriteReviewQueueSnapshot{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var queue WriteReviewQueueSnapshot
		found, err := fsutil.ReadJSON(filepath.Join(dir, entry.Name()), &queue)
		if err != nil {
			return nil, err
		}
		if found && queue.ID != "" {
			queues = append(queues, queue)
		}
	}
	sort.SliceStable(queues, func(i, j int) bool {
		left := parseTimelineTime(queues[i].CreatedAt)
		right := parseTimelineTime(queues[j].CreatedAt)
		if !left.Equal(right) {
			return left.After(right)
		}
		return queues[i].ID > queues[j].ID
	})
	return queues, nil
}

func writeReviewQueueMatchesAdmission(queue WriteReviewQueueSnapshot, admission WriteAdmissionEntry) bool {
	if queue.AdmissionID != "" && queue.AdmissionID == admission.ID {
		return true
	}
	if queue.DeploymentExecutionID != "" && admission.OperationType == "deployment_execution" && queue.DeploymentExecutionID == admission.OperationID {
		return true
	}
	return false
}

func sortWriteReviewPackets(packets []WriteReviewPacket) {
	sort.SliceStable(packets, func(i, j int) bool {
		left := parseTimelineTime(packets[i].CreatedAt)
		right := parseTimelineTime(packets[j].CreatedAt)
		if !left.Equal(right) {
			return left.After(right)
		}
		return packets[i].ID > packets[j].ID
	})
}

func renderWriteReviewPacketMarkdown(packet WriteReviewPacket) string {
	var builder strings.Builder
	builder.WriteString("# Write Review Packet\n\n")
	builder.WriteString("- Packet: `" + packet.ID + "`\n")
	builder.WriteString("- Operation: `" + packet.OperationType + "/" + packet.OperationID + "`\n")
	builder.WriteString("- Status: `" + packet.Status + "`\n")
	builder.WriteString("- Decision: `" + packet.Decision + "`\n")
	if packet.Provider != "" {
		builder.WriteString("- Provider: `" + packet.Provider + "`\n")
	}
	if packet.Environment != "" {
		builder.WriteString("- Environment: `" + packet.Environment + "`\n")
	}
	if packet.AdmissionID != "" {
		builder.WriteString("- Admission: `" + packet.AdmissionID + "`\n")
	}
	if packet.RemoteRehearsalID != "" {
		builder.WriteString("- Remote rehearsal: `" + packet.RemoteRehearsalID + "` (" + packet.RemoteRehearsalDecision + ")\n")
	}
	if len(packet.QueueItemIDs) > 0 {
		builder.WriteString("- Queue items: `" + strings.Join(packet.QueueItemIDs, "`, `") + "`\n")
	}
	if len(packet.Reasons) > 0 {
		builder.WriteString("\n## Reasons\n")
		for _, reason := range packet.Reasons {
			builder.WriteString("- `" + reason + "`\n")
		}
	}
	if len(packet.RuleRefs) > 0 {
		builder.WriteString("\n## Rule Refs\n")
		for _, ref := range packet.RuleRefs {
			builder.WriteString("- `" + ref + "`\n")
		}
	}
	if len(packet.EvidenceRefs) > 0 {
		builder.WriteString("\n## Evidence Refs\n")
		for _, ref := range packet.EvidenceRefs {
			builder.WriteString("- `" + ref + "`\n")
		}
	}
	return builder.String()
}

func finishWriteReviewPacket(rootDir string, packet WriteReviewPacket) (WriteReviewPacket, error) {
	packet = normalizeWriteReviewPacket(packet)
	if err := fsutil.EnsureDir(writeReviewPacketDir(rootDir)); err != nil {
		return WriteReviewPacket{}, err
	}
	if err := fsutil.WriteJSON(writeReviewPacketPath(rootDir, packet.ID), packet); err != nil {
		return WriteReviewPacket{}, err
	}
	if err := fsutil.AppendJSONL(filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "write-review-packets.jsonl"), packet); err != nil {
		return WriteReviewPacket{}, err
	}
	_ = logging.Log(rootDir, "release", "operations.write_review_packet.created", map[string]any{
		"packet_id":    packet.ID,
		"admission_id": packet.AdmissionID,
		"operation_id": packet.OperationID,
		"provider":     packet.Provider,
		"environment":  packet.Environment,
		"status":       packet.Status,
		"decision":     packet.Decision,
	})
	if _, err := evidence.Add(rootDir, evidence.AddOptions{
		ParentType:  "write_review_packet",
		ParentID:    packet.ID,
		SubjectType: packet.OperationType,
		SubjectID:   packet.OperationID,
		Operation:   "operations.write_review_packet.create",
		Status:      packet.Status,
		Decision:    packet.Decision,
		Reasons:     packet.Reasons,
		Source:      "operations",
		Artifacts: []evidence.ArtifactRef{{
			Kind: "write_review_packet",
			ID:   packet.ID,
			Path: filepath.ToSlash(filepath.Join(".moyuan", "lifecycle", "deployments", "write-review-packets", packet.ID+".json")),
		}},
	}); err != nil {
		return WriteReviewPacket{}, err
	}
	return packet, nil
}

func writeReviewPacketDir(rootDir string) string {
	return filepath.Join(workspace.ForRoot(rootDir).DeploymentsDir, "write-review-packets")
}

func writeReviewPacketPath(rootDir string, id string) string {
	return filepath.Join(writeReviewPacketDir(rootDir), id+".json")
}
