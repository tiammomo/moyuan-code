"use client";

import {
  Activity,
  AlertTriangle,
  Boxes,
  Brain,
  CheckCircle2,
  ChevronRight,
  CircleDotDashed,
  GitBranch,
  Layers3,
  Lock,
  MemoryStick,
  Network,
  Play,
  Rocket,
  Search,
  Server,
  ShieldCheck,
  Sparkles,
  TerminalSquare,
} from "lucide-react";
import { useMemo, useState, type FormEvent } from "react";
import type { ConsoleSnapshot, IssueNode, StatusTone } from "@/lib/types";

const laneLabels: Record<IssueNode["lane"], string> = {
  plan: "Planning",
  backend: "Backend",
  frontend: "Frontend",
  quality: "Quality",
  release: "Release",
};

const nav = [
  { label: "Projects", icon: Boxes },
  { label: "Issue Graph", icon: Network },
  { label: "Runs", icon: TerminalSquare },
  { label: "Quality", icon: ShieldCheck },
  { label: "Memory", icon: Brain },
  { label: "Providers", icon: Sparkles },
  { label: "Deployments", icon: Rocket },
  { label: "Audit", icon: Lock },
];

export function ConsoleWorkbench({ snapshot }: { snapshot: ConsoleSnapshot }) {
  const [selectedIssueID, setSelectedIssueID] = useState(snapshot.issues[0]?.id ?? "");
  const [requirementText, setRequirementText] = useState("");
  const [requirementState, setRequirementState] = useState<RequirementSubmitState>({ status: "idle" });
  const [visualActionState, setVisualActionState] = useState<Record<string, VisualActionState>>({});
  const selectedIssue = snapshot.issues.find((issue) => issue.id === selectedIssueID) ?? snapshot.issues[0];
  const groupedIssues = useMemo(() => groupIssues(snapshot.issues), [snapshot.issues]);

  async function submitRequirement(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const text = requirementText.trim();
    if (!text) {
      setRequirementState({ status: "error", message: "Requirement text is required." });
      return;
    }
    setRequirementState({ status: "planning", message: "Planning issue graph..." });
    try {
      const response = await fetch(`/api/projects/${snapshot.project.id}/requirements/plan`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ text }),
      });
      const payload = (await response.json()) as RequirementPlanEnvelope;
      if (!payload.requirement) {
        throw new Error(payload.error ?? "Requirement planner returned no result.");
      }
      const decision = payload.requirement.clarification_decision;
      const needsInput = Boolean(decision?.required);
      setRequirementState({
        status: needsInput ? "needs_user_input" : "planned",
        id: payload.requirement.id,
        epic: payload.requirement.epic_id,
        message: needsInput
          ? decision?.questions?.[0] ?? "This requirement needs clarification."
          : `${payload.requirement.issues?.length ?? 0} issues generated.`,
      });
    } catch (error) {
      setRequirementState({ status: "error", message: error instanceof Error ? error.message : "Requirement planning failed." });
    }
  }

  async function runVisualDryRun(assetID: string) {
    setVisualActionState((current) => ({
      ...current,
      [assetID]: { status: "running", message: "Creating dry-run render..." },
    }));
    try {
      const response = await fetch(`/api/projects/${snapshot.project.id}/visuals/assets/${encodeURIComponent(assetID)}/render`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ mode: "dry_run" }),
      });
      const payload = (await response.json()) as VisualRenderEnvelope;
      const execution = payload.visual_render_execution;
      if (!response.ok || !execution) {
        throw new Error(payload.error ?? "Visual render dry-run failed.");
      }
      setVisualActionState((current) => ({
        ...current,
        [assetID]: {
          status: execution.status === "completed" ? "completed" : "blocked",
          executionID: execution.id,
          message: `${execution.decision ?? execution.status ?? "dry-run recorded"}`,
        },
      }));
    } catch (error) {
      setVisualActionState((current) => ({
        ...current,
        [assetID]: {
          status: "error",
          message: error instanceof Error ? error.message : "Visual render dry-run failed.",
        },
      }));
    }
  }

  return (
    <main className="shell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brandMark">
            <Layers3 size={20} />
          </div>
          <div>
            <strong>Moyuan</strong>
            <span>Control Console</span>
          </div>
        </div>

        <nav className="navList">
          {nav.map((item) => (
            <button className="navItem" key={item.label} type="button">
              <item.icon size={17} />
              <span>{item.label}</span>
            </button>
          ))}
        </nav>

        <div className="sideCard">
          <div className="sideCardTop">
            <span>Runtime</span>
            <StatusPill tone={snapshot.backendStatus} label={snapshot.mode === "live" ? "live" : "demo"} />
          </div>
          <strong>3000 &gt; 8080</strong>
          <small>Next.js 16 / Go Gin</small>
        </div>
      </aside>

      <section className="workspace">
        <header className="topbar">
          <div>
            <p className="eyebrow">AI Engineering Ops</p>
            <h1>{snapshot.project.name}</h1>
          </div>
          <div className="topActions">
            <div className="searchBox">
              <Search size={16} />
              <span>Jump to issue, run, provider...</span>
            </div>
            <button className="iconButton" type="button" aria-label="Run selected issue">
              <Play size={18} />
            </button>
          </div>
        </header>

        <section className="heroGrid">
          <MetricCard label="Issues" value={snapshot.stats.issues} tone="neutral" detail={`${snapshot.stats.accepted} accepted`} />
          <MetricCard label="Blocked" value={snapshot.stats.blocked} tone={snapshot.stats.blocked > 0 ? "warning" : "ok"} detail="dependency / approval" />
          <MetricCard label="Providers" value={snapshot.stats.providers} tone="running" detail="Claude / Codex / API" />
          <MetricCard label="Deploys" value={snapshot.stats.executions} tone="neutral" detail={`${snapshot.stats.deployments} plans`} />
        </section>

        <section className="opsGrid">
          <div className="panel requirementPanel">
            <PanelTitle icon={<Sparkles size={18} />} title="Requirement Intake" meta="plan to issue graph" />
            <form className="requirementForm" onSubmit={submitRequirement}>
              <textarea
                aria-label="Requirement text"
                onChange={(event) => setRequirementText(event.target.value)}
                placeholder="Describe a feature, fix, or operational change with verification expectations..."
                rows={3}
                value={requirementText}
              />
              <div className="formFooter">
                <button className="primaryButton" disabled={requirementState.status === "planning"} type="submit">
                  <Play size={16} />
                  <span>{requirementState.status === "planning" ? "Planning" : "Plan Issues"}</span>
                </button>
                {requirementState.status !== "idle" ? (
                  <div className={`formResult ${requirementState.status}`}>
                    <strong>{requirementState.status.replaceAll("_", " ")}</strong>
                    <span>{requirementState.message}</span>
                    {requirementState.epic ? <code>{requirementState.epic}</code> : null}
                  </div>
                ) : null}
              </div>
            </form>
          </div>

          <div className="panel executionPanel">
            <PanelTitle icon={<Rocket size={18} />} title="Deployment Executions" meta={`${snapshot.executions.length} recent`} />
            <div className="executionList">
              {snapshot.executions.length > 0 ? (
                snapshot.executions.map((execution) => (
                  <div className="executionItem" key={execution.id}>
                    <div>
                      <strong>{execution.mode}</strong>
                      <span>{execution.decision}</span>
                    </div>
                    <StatusPill tone={toneForStatus(execution.status)} label={execution.status} />
                  </div>
                ))
              ) : snapshot.deployments.length > 0 ? (
                snapshot.deployments.map((deployment) => (
                  <div className="executionItem" key={deployment.id}>
                    <div>
                      <strong>{deployment.environment}</strong>
                      <span>{deployment.decision}</span>
                    </div>
                    <StatusPill tone={toneForStatus(deployment.status)} label={deployment.status} />
                  </div>
                ))
              ) : (
                <div className="emptyState">No deployment executions yet</div>
              )}
            </div>
          </div>
        </section>

        <section className="mainGrid">
          <div className="panel graphPanel">
            <PanelTitle icon={<Network size={18} />} title="Issue Graph" meta="dependency aware" />
            <div className="graphCanvas">
              {Object.entries(groupedIssues).map(([lane, issues]) => (
                <div className="lane" key={lane}>
                  <div className="laneTitle">{laneLabels[lane as IssueNode["lane"]]}</div>
                  <div className="laneNodes">
                    {issues.map((issue) => (
                      <button
                        className={`issueNode ${statusClass(issue.status)} ${selectedIssue?.id === issue.id ? "selected" : ""}`}
                        key={issue.id}
                        onClick={() => setSelectedIssueID(issue.id)}
                        type="button"
                      >
                        <span>{issue.title}</span>
                        <small>{issue.role}</small>
                      </button>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <aside className="panel inspector">
            <PanelTitle icon={<CircleDotDashed size={18} />} title="Inspector" meta={selectedIssue?.id ?? "issue"} />
            {selectedIssue ? (
              <div className="inspectorBody">
                <h2>{selectedIssue.title}</h2>
                <StatusPill tone={toneForStatus(selectedIssue.status)} label={selectedIssue.status} />
                <dl>
                  <div>
                    <dt>Run</dt>
                    <dd>{selectedIssue.run_id ?? "not started"}</dd>
                  </div>
                  <div>
                    <dt>Subagent</dt>
                    <dd>{selectedIssue.subagent_id ?? "not assigned"}</dd>
                  </div>
                  <div>
                    <dt>Role</dt>
                    <dd>{selectedIssue.role}</dd>
                  </div>
                  <div>
                    <dt>Runtime</dt>
                    <dd>{selectedIssue.runtime ?? "pending"}</dd>
                  </div>
                  <div>
                    <dt>Runtime Status</dt>
                    <dd>{selectedIssue.runtime_status ?? "pending"}</dd>
                  </div>
                  <div>
                    <dt>Provider</dt>
                    <dd>{selectedIssue.provider ?? "route pending"}</dd>
                  </div>
                  <div>
                    <dt>Quality</dt>
                    <dd>{selectedIssue.quality ?? "not started"}</dd>
                  </div>
                  <div>
                    <dt>Review</dt>
                    <dd>{selectedIssue.review_status ?? "not reviewed"}</dd>
                  </div>
                  <div>
                    <dt>Quality Report</dt>
                    <dd>{selectedIssue.quality_report_id ?? "none"}</dd>
                  </div>
                </dl>
                {selectedIssue.quality_decision ? (
                  <div className="decisionStrip">
                    <ShieldCheck size={16} />
                    <div>
                      <strong>{selectedIssue.quality_decision}</strong>
                      <span>{selectedIssue.quality_reasons?.[0] ?? "quality explanation available"}</span>
                    </div>
                  </div>
                ) : null}
                {selectedIssue.skills && selectedIssue.skills.length > 0 ? (
                  <div className="chipSection">
                    <span>Skills</span>
                    <div className="chipList">
                      {selectedIssue.skills.map((skill) => (
                        <code key={skill}>{skill}</code>
                      ))}
                    </div>
                  </div>
                ) : null}
                {selectedIssue.output_contract && selectedIssue.output_contract.length > 0 ? (
                  <div className="chipSection">
                    <span>Output Contract</span>
                    <div className="chipList">
                      {selectedIssue.output_contract.map((item) => (
                        <code key={item}>{item}</code>
                      ))}
                    </div>
                  </div>
                ) : null}
                {selectedIssue.depends_on && selectedIssue.depends_on.length > 0 ? (
                  <div className="dependencyList">
                    {selectedIssue.depends_on.map((dependency) => (
                      <span key={dependency}>
                        <ChevronRight size={13} />
                        {dependency}
                      </span>
                    ))}
                  </div>
                ) : null}
                {selectedIssue.quality_reasons && selectedIssue.quality_reasons.length > 1 ? (
                  <div className="reasonList">
                    {selectedIssue.quality_reasons.slice(1).map((reason) => (
                      <span key={reason}>{reason}</span>
                    ))}
                  </div>
                ) : null}
                {selectedIssue.blocking_findings && selectedIssue.blocking_findings.length > 0 ? (
                  <div className="findingList">
                    {selectedIssue.blocking_findings.map((finding) => (
                      <div key={finding.id}>
                        <strong>
                          {finding.severity} / {finding.category}
                        </strong>
                        <span>{finding.message}</span>
                        {finding.path ? <code>{finding.path}</code> : null}
                      </div>
                    ))}
                  </div>
                ) : null}
                {selectedIssue.blocked_reason ? <div className="warningLine">{selectedIssue.blocked_reason}</div> : null}
              </div>
            ) : null}
          </aside>
        </section>

        <section className="lowerGrid">
          <div className="panel">
            <PanelTitle icon={<Activity size={18} />} title="Run Timeline" meta={`${snapshot.runs.length} runs`} />
            <div className="timeline">
              {snapshot.timeline.map((event) => (
                <div className="timelineItem" key={event.id}>
                  <StatusDot tone={event.tone} />
                  <div>
                    <strong>{event.title}</strong>
                    <span>{event.detail}</span>
                  </div>
                  <time>{event.time}</time>
                </div>
              ))}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<ShieldCheck size={18} />} title="Quality Gates" meta="diff-first" />
            <div className="qualityList">
              {snapshot.quality.map((item) => (
                <div className="qualityItem" key={item.id}>
                  {item.severity === "ok" ? <CheckCircle2 size={18} /> : <AlertTriangle size={18} />}
                  <div>
                    <strong>{item.title}</strong>
                    <span>{item.detail}</span>
                  </div>
                  <StatusPill tone={item.severity} label={item.status} />
                </div>
              ))}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<MemoryStick size={18} />} title="Memory" meta="compact aware" />
            <div className="memoryList">
              {snapshot.memory.map((record) => (
                <div className="memoryItem" key={record.id}>
                  <span>{record.kind}</span>
                  <strong>{record.summary}</strong>
                  <meter value={record.score} min="0" max="1" />
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="observabilityGrid">
          <div className="panel">
            <PanelTitle icon={<TerminalSquare size={18} />} title="Runtime Recoveries" meta={`${snapshot.runtime_recoveries.length} archived`} />
            <div className="signalList">
              {snapshot.runtime_recoveries.length > 0 ? (
                snapshot.runtime_recoveries.map((recovery) => (
                  <div className="signalItem" key={recovery.id}>
                    <div className="signalHeader">
                      <strong>{compactID(recovery.issue_id || recovery.run_id || recovery.id)}</strong>
                      <StatusPill tone={toneForStatus(recovery.status)} label={recovery.status} />
                    </div>
                    <span>
                      {recovery.failure_category} / {recovery.runtime_id || "runtime pending"}
                    </span>
                    <div className="signalMeta">
                      {recovery.fallback_candidate ? <code>fallback {recovery.fallback_candidate}</code> : null}
                      {recovery.native_session_id ? <code>{compactID(recovery.native_session_id)}</code> : null}
                      {recovery.diff_summary_path ? <code>{shortPath(recovery.diff_summary_path)}</code> : null}
                    </div>
                  </div>
                ))
              ) : (
                <div className="emptyState">No runtime recovery archived</div>
              )}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<CircleDotDashed size={18} />} title="Subagent Backlog" meta={`${snapshot.subagent_backlog.length} waiting`} />
            <div className="signalList">
              {snapshot.subagent_backlog.length > 0 ? (
                snapshot.subagent_backlog.map((item) => (
                  <div className="signalItem" key={`${item.issue_id}-${item.subagent_id}`}>
                    <div className="signalHeader">
                      <strong>{compactID(item.issue_id)}</strong>
                      <StatusPill tone={toneForStatus(item.status)} label={item.status} />
                    </div>
                    <span>{item.reason || item.failure_category || "waiting for scheduler decision"}</span>
                    <div className="signalMeta">
                      <code>{compactID(item.subagent_id)}</code>
                      <code>
                        retry {item.retry_count}/{item.max_retries}
                      </code>
                      {item.recovery_id ? <code>{compactID(item.recovery_id)}</code> : null}
                    </div>
                  </div>
                ))
              ) : (
                <div className="emptyState">No subagent backlog</div>
              )}
            </div>
          </div>

          <div className="panel">
            <PanelTitle
              icon={<Sparkles size={18} />}
              title="Visual Assets"
              meta={`${snapshot.visual_assets.length} plans / ${snapshot.visual_render_executions.length} renders`}
            />
            <div className="signalList">
              {snapshot.visual_assets.length > 0 || snapshot.visual_render_executions.length > 0 ? (
                <>
                  {snapshot.visual_assets.map((asset) => {
                    const actionState = visualActionState[asset.id];
                    return (
                      <div className="signalItem" key={asset.id}>
                        <div className="signalHeader">
                          <strong>{asset.title}</strong>
                          <StatusPill tone={toneForStatus(asset.status)} label={asset.status} />
                        </div>
                        <span>
                          {asset.diagram_type} / {asset.size}
                        </span>
                        <div className="signalMeta">
                          {asset.provider_id ? <code>{asset.provider_id}</code> : null}
                          {asset.model_id ? <code>{asset.model_id}</code> : null}
                          <code>{shortPath(asset.prompt_path || asset.spec_path)}</code>
                        </div>
                        {asset.route_reason ? <small>{asset.route_reason}</small> : null}
                        <div className="signalActions">
                          <button
                            className="inlineActionButton"
                            disabled={actionState?.status === "running"}
                            onClick={() => void runVisualDryRun(asset.id)}
                            type="button"
                          >
                            <Play size={13} />
                            <span>{actionState?.status === "running" ? "Running" : "Dry Run"}</span>
                          </button>
                          {actionState ? (
                            <small className={`actionMessage ${actionState.status}`}>
                              {actionState.executionID ? `${compactID(actionState.executionID)} / ` : ""}
                              {actionState.message}
                            </small>
                          ) : null}
                        </div>
                      </div>
                    );
                  })}
                  {snapshot.visual_render_executions.map((execution) => (
                    <div className="signalItem" key={execution.id}>
                      <div className="signalHeader">
                        <strong>{execution.title || compactID(execution.asset_id || execution.id)}</strong>
                        <StatusPill tone={toneForStatus(execution.status)} label={execution.mode} />
                      </div>
                      <span>
                        {execution.decision} / {execution.step_count} steps
                      </span>
                      <div className="signalMeta">
                        <code>{execution.status}</code>
                        {execution.provider_id ? <code>{execution.provider_id}</code> : null}
                        {execution.script_path ? <code>{shortPath(execution.script_path)}</code> : null}
                        {execution.image_path ? <code>{shortPath(execution.image_path)}</code> : null}
                      </div>
                      {execution.reasons[0] ? <small>{execution.reasons[0]}</small> : null}
                    </div>
                  ))}
                </>
              ) : (
                <div className="emptyState">No visual assets planned</div>
              )}
            </div>
          </div>
        </section>

        <section className="bottomGrid">
          <div className="panel">
            <PanelTitle icon={<Sparkles size={18} />} title="Providers & Runtimes" meta={`${snapshot.providers.length} registered`} />
            <div className="providerMatrix">
              {snapshot.providers.map((provider) => (
                <div className="providerRow" key={provider.id}>
                  <StatusDot tone={provider.enabled ? "ok" : "neutral"} />
                  <div>
                    <strong>{provider.name}</strong>
                    <span>
                      {provider.vendor} / {provider.runtime_id || provider.api_type}
                    </span>
                  </div>
                  <code>{provider.model || provider.id}</code>
                </div>
              ))}
            </div>
          </div>

          <div className="panel">
            <PanelTitle icon={<Server size={18} />} title="Server Resources" meta="test_dev / production" />
            <div className="resourceList">
              {snapshot.resources.length > 0 ? (
                snapshot.resources.map((resource) => (
                  <div className="resourceItem" key={resource.id}>
                    <div>
                      <strong>{resource.id}</strong>
                      <span>
                        {resource.host}
                        {resource.health ? ` / ${resource.health}` : ""}
                      </span>
                    </div>
                    <StatusPill tone={resource.environment === "production" ? "warning" : "ok"} label={resource.environment} />
                  </div>
                ))
              ) : (
                <div className="emptyState">No resources registered</div>
              )}
            </div>
          </div>

          <div className="panel releasePanel">
            <PanelTitle icon={<GitBranch size={18} />} title="Release Pipeline" meta="GitHub / Gitee" />
            <div className="releaseSteps">
              <span>accepted issues</span>
              <ChevronRight size={15} />
              <span>release branch</span>
              <ChevronRight size={15} />
              <span>tag + PR/MR</span>
              <ChevronRight size={15} />
              <span>deploy plan</span>
            </div>
            <div className="approvalStrip">
              <Lock size={16} />
              <span>production deploy requires approval, smoke, monitor and rollback plan</span>
            </div>
          </div>
        </section>
      </section>
    </main>
  );
}

type RequirementSubmitState = {
  status: "idle" | "planning" | "planned" | "needs_user_input" | "error";
  id?: string;
  epic?: string;
  message?: string;
};

type RequirementPlanEnvelope = {
  error?: string;
  requirement?: {
    id?: string;
    epic_id?: string;
    issues?: unknown[];
    clarification_decision?: {
      required?: boolean;
      questions?: string[];
    };
  };
};

type VisualActionState = {
  status: "running" | "completed" | "blocked" | "error";
  executionID?: string;
  message?: string;
};

type VisualRenderEnvelope = {
  error?: string;
  visual_render_execution?: {
    id?: string;
    status?: string;
    decision?: string;
  };
};

function PanelTitle({ icon, title, meta }: { icon: React.ReactNode; title: string; meta: string }) {
  return (
    <div className="panelTitle">
      <div>
        {icon}
        <strong>{title}</strong>
      </div>
      <span>{meta}</span>
    </div>
  );
}

function MetricCard({
  label,
  value,
  tone,
  detail,
}: {
  label: string;
  value: number;
  tone: StatusTone;
  detail: string;
}) {
  return (
    <div className={`metricCard ${tone}`}>
      <span>{label}</span>
      <strong>{value}</strong>
      <small>{detail}</small>
    </div>
  );
}

function StatusPill({ tone, label }: { tone: StatusTone; label: string }) {
  return <span className={`statusPill ${tone}`}>{label}</span>;
}

function StatusDot({ tone }: { tone: StatusTone }) {
  return <span className={`statusDot ${tone}`} />;
}

function groupIssues(issues: IssueNode[]) {
  return issues.reduce<Record<IssueNode["lane"], IssueNode[]>>(
    (acc, issue) => {
      acc[issue.lane].push(issue);
      return acc;
    },
    { plan: [], backend: [], frontend: [], quality: [], release: [] },
  );
}

function toneForStatus(status: string): StatusTone {
  if (status === "accepted" || status === "passed" || status === "ready" || status === "completed" || status === "planned") return "ok";
  if (status === "running" || status === "dispatch" || status === "retrying") return "running";
  if (status === "blocked" || status === "rejected" || status === "failed" || status === "route_blocked") return "blocked";
  if (status === "waiting" || status === "pending" || status === "archived") return "warning";
  return "neutral";
}

function statusClass(status: string) {
  return toneForStatus(status);
}

function compactID(value: string) {
  if (!value) return "unknown";
  if (value.length <= 28) return value;
  return `${value.slice(0, 18)}...${value.slice(-7)}`;
}

function shortPath(value?: string) {
  if (!value) return "path pending";
  const parts = value.split("/").filter(Boolean);
  return parts.slice(-3).join("/");
}
