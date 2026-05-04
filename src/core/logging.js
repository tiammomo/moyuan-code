import path from 'node:path';
import { appendJsonl, readJsonl, tailLines } from '../lib/fs.js';
import { getWorkspacePaths } from './paths.js';

function streamFile(rootDir, stream) {
  const { logsDir } = getWorkspacePaths(rootDir);
  return path.join(logsDir, `${stream}.jsonl`);
}

export async function logEvent(rootDir, stream, event, data = {}) {
  const record = {
    ts: new Date().toISOString(),
    stream,
    event,
    ...data
  };

  await appendJsonl(streamFile(rootDir, stream), record);
  return record;
}

export async function readEvents(rootDir, stream) {
  return readJsonl(streamFile(rootDir, stream));
}

export async function tailEvents(rootDir, stream, limit = 20) {
  return tailLines(streamFile(rootDir, stream), limit);
}
