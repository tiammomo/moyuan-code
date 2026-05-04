import path from 'node:path';
import { appendJsonl, fileExists, readJson, writeJson } from '../lib/fs.js';
import { getWorkspacePaths } from './paths.js';
import { slugify } from '../lib/text.js';
import { logEvent } from './logging.js';

function phase1Template() {
  return {
    epic: {
      id: 'phase1-epic',
      title: 'local-cli-mvp',
      status: 'planned'
    },
    nodes: [
      { id: 'phase1-001', title: 'workspace-core', status: 'ready', depends_on: [] },
      { id: 'phase1-002', title: 'auth-context', status: 'blocked', depends_on: ['phase1-001'] },
      { id: 'phase1-003', title: 'logging-audit', status: 'blocked', depends_on: ['phase1-001'] },
      { id: 'phase1-004', title: 'cli-bootstrap', status: 'blocked', depends_on: ['phase1-001', 'phase1-002', 'phase1-003'] },
      { id: 'phase1-005', title: 'git-adapter-basics', status: 'blocked', depends_on: ['phase1-001', 'phase1-002', 'phase1-003'] },
      { id: 'phase1-006', title: 'runtime-adapters-core', status: 'blocked', depends_on: ['phase1-001', 'phase1-003'] },
      { id: 'phase1-007', title: 'project-comprehension', status: 'blocked', depends_on: ['phase1-005'] },
      { id: 'phase1-008', title: 'orchestrator-core', status: 'blocked', depends_on: ['phase1-004', 'phase1-005', 'phase1-006', 'phase1-007'] },
      { id: 'phase1-009', title: 'scheduler-core', status: 'blocked', depends_on: ['phase1-008'] },
      { id: 'phase1-010', title: 'quality-gates-core', status: 'blocked', depends_on: ['phase1-003', 'phase1-005', 'phase1-006'] },
      { id: 'phase1-011', title: 'memory-basics', status: 'blocked', depends_on: ['phase1-007', 'phase1-008'] },
      { id: 'phase1-012', title: 'repair-basics', status: 'blocked', depends_on: ['phase1-010', 'phase1-011'] },
      { id: 'phase1-013', title: 'e2e-smoke', status: 'blocked', depends_on: ['phase1-004', 'phase1-005', 'phase1-006', 'phase1-007', 'phase1-008', 'phase1-009', 'phase1-010', 'phase1-011', 'phase1-012'] }
    ]
  };
}

function graphPath(rootDir, epicId) {
  return path.join(getWorkspacePaths(rootDir).issueGraphsDir, `${slugify(epicId)}.json`);
}

function schedulePath(rootDir, epicId) {
  return path.join(getWorkspacePaths(rootDir).schedulesDir, `${slugify(epicId)}.json`);
}

export async function loadIssueGraph(rootDir, epicId) {
  const file = graphPath(rootDir, epicId);
  if (await fileExists(file)) {
    return readJson(file, null);
  }

  if (slugify(epicId) === 'phase1-epic' || epicId === 'phase1-epic') {
    return phase1Template();
  }

  return null;
}

export async function saveIssueGraph(rootDir, epicId, graph) {
  const file = graphPath(rootDir, epicId);
  await writeJson(file, graph);
  await logEvent(rootDir, 'run', 'issue.graph.saved', { epic_id: epicId, path: file });
  return graph;
}

export async function summarizeIssueGraph(graph) {
  if (!graph) {
    return {
      nodes: [],
      ready_queue: [],
      blocked_queue: []
    };
  }

  const nodes = graph.nodes ?? [];
  const ready_queue = nodes.filter((node) => node.status === 'ready').map((node) => node.id);
  const blocked_queue = nodes.filter((node) => node.status === 'blocked').map((node) => node.id);
  return {
    ...graph,
    ready_queue,
    blocked_queue
  };
}

export async function loadSchedule(rootDir, epicId) {
  const file = schedulePath(rootDir, epicId);
  if (await fileExists(file)) {
    return readJson(file, null);
  }

  const graph = await loadIssueGraph(rootDir, epicId);
  if (!graph) {
    return null;
  }

  return summarizeIssueGraph(graph);
}

export async function saveSchedule(rootDir, epicId, schedule) {
  const file = schedulePath(rootDir, epicId);
  await writeJson(file, schedule);
  await logEvent(rootDir, 'run', 'issue.schedule.saved', { epic_id: epicId, path: file });
  return schedule;
}

export async function generatePhase1IssueGraph(rootDir) {
  const graph = phase1Template();
  await saveIssueGraph(rootDir, graph.epic.id, graph);
  const schedule = await summarizeIssueGraph(graph);
  await saveSchedule(rootDir, graph.epic.id, schedule);
  return { graph, schedule };
}
