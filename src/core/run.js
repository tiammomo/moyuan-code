import path from 'node:path';
import crypto from 'node:crypto';
import { appendJsonl, fileExists, readJson, writeJson } from '../lib/fs.js';
import { getWorkspacePaths } from './paths.js';
import { logEvent } from './logging.js';

function runPath(rootDir, runId) {
  return path.join(getWorkspacePaths(rootDir).runsDir, `${runId}.json`);
}

function makeRunId(taskId) {
  const suffix = crypto.randomBytes(3).toString('hex');
  return `run-${taskId}-${Date.now()}-${suffix}`;
}

export async function createRun(rootDir, taskId, payload = {}) {
  const runId = payload.runId ?? makeRunId(taskId);
  const record = {
    id: runId,
    task_id: taskId,
    status: 'queued',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    payload
  };

  await writeJson(runPath(rootDir, runId), record);
  await appendJsonl(path.join(getWorkspacePaths(rootDir).runsDir, 'events.jsonl'), {
    ts: new Date().toISOString(),
    event: 'run.created',
    run_id: runId,
    task_id: taskId
  });
  await logEvent(rootDir, 'run', 'run.created', { run_id: runId, task_id: taskId });
  return record;
}

export async function readRun(rootDir, runId) {
  const file = runPath(rootDir, runId);
  if (!(await fileExists(file))) {
    return null;
  }
  return readJson(file, null);
}
