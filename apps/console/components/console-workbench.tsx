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
import { useMemo, useState } from "react";
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
  const selectedIssue = snapshot.issues.find((issue) => issue.id === selectedIssueID) ?? snapshot.issues[0];
  const groupedIssues = useMemo(() => groupIssues(snapshot.issues), [snapshot.issues]);

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
          <MetricCard label="Resources" value={snapshot.stats.resources} tone="neutral" detail="test_dev / production" />
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
                    <dt>Role</dt>
                    <dd>{selectedIssue.role}</dd>
                  </div>
                  <div>
                    <dt>Runtime</dt>
                    <dd>{selectedIssue.runtime ?? "pending"}</dd>
                  </div>
                  <div>
                    <dt>Provider</dt>
                    <dd>{selectedIssue.provider ?? "route pending"}</dd>
                  </div>
                  <div>
                    <dt>Quality</dt>
                    <dd>{selectedIssue.quality ?? "not started"}</dd>
                  </div>
                </dl>
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
                {selectedIssue.blocked_reason ? <div className="warningLine">{selectedIssue.blocked_reason}</div> : null}
              </div>
            ) : null}
          </aside>
        </section>

        <section className="lowerGrid">
          <div className="panel">
            <PanelTitle icon={<Activity size={18} />} title="Run Timeline" meta="live workbench" />
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
                      <span>{resource.host}</span>
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
  if (status === "accepted" || status === "passed" || status === "ready") return "ok";
  if (status === "running" || status === "dispatch") return "running";
  if (status === "blocked" || status === "rejected") return "blocked";
  if (status === "waiting" || status === "pending") return "warning";
  return "neutral";
}

function statusClass(status: string) {
  return toneForStatus(status);
}
