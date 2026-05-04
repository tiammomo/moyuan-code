import path from 'node:path';
import { appendJsonl, fileExists, readJson, readYaml, writeJson, writeText } from '../lib/fs.js';
import { getWorkspacePaths } from './paths.js';
import { runShell } from '../lib/process.js';
import { logEvent } from './logging.js';

function reportPaths(rootDir, id) {
  const base = getWorkspacePaths(rootDir).qualityDir;
  return {
    json: path.join(base, 'reports', `${id}.json`),
    md: path.join(base, 'reports', `${id}.md`)
  };
}

async function getCommandLists(rootDir) {
  const project = await readYaml(getWorkspacePaths(rootDir).projectYaml, null);
  const stack = project?.stack ?? {};
  return {
    build: stack.build_commands ?? [],
    lint: stack.lint_commands ?? [],
    test: stack.test_commands ?? []
  };
}

async function runChecks(rootDir, commandList, type) {
  const results = [];
  for (const command of commandList) {
    const startedAt = new Date().toISOString();
    const result = await runShell(command, { cwd: rootDir });
    results.push({
      type,
      command,
      started_at: startedAt,
      finished_at: new Date().toISOString(),
      status: result.code === 0 ? 'passed' : 'failed',
      exit_code: result.code,
      stdout: result.stdout,
      stderr: result.stderr
    });
  }
  return results;
}

export async function runQualityCheck(rootDir, taskId, options = {}) {
  const commands = await getCommandLists(rootDir);
  const reportId = options.reportId ?? `quality-${taskId}-${Date.now()}`;
  const paths = reportPaths(rootDir, reportId);

  const checks = [];
  if (commands.build.length) {
    checks.push(...(await runChecks(rootDir, commands.build, 'build')));
  } else {
    checks.push({ type: 'build', command: null, status: 'skipped', reason: 'no build command configured' });
  }
  if (commands.lint.length) {
    checks.push(...(await runChecks(rootDir, commands.lint, 'lint')));
  } else {
    checks.push({ type: 'lint', command: null, status: 'skipped', reason: 'no lint command configured' });
  }
  if (commands.test.length) {
    checks.push(...(await runChecks(rootDir, commands.test, 'test')));
  } else {
    checks.push({ type: 'test', command: null, status: 'skipped', reason: 'no test command configured' });
  }

  const failed = checks.some((check) => check.status === 'failed');
  const report = {
    id: reportId,
    task_id: taskId,
    created_at: new Date().toISOString(),
    status: failed ? 'failed' : 'passed',
    checks
  };

  await writeJson(paths.json, report);
  await writeText(
    paths.md,
    [
      `# Quality Report`,
      '',
      `- Task ID: \`${taskId}\``,
      `- Report ID: \`${reportId}\``,
      `- Status: \`${report.status}\``,
      '',
      ...checks.map((check) => `- ${check.type}: ${check.status}${check.command ? ` (${check.command})` : ''}`)
    ].join('\n') + '\n'
  );

  await appendJsonl(path.join(getWorkspacePaths(rootDir).qualityDir, 'events.jsonl'), {
    ts: new Date().toISOString(),
    task_id: taskId,
    report_id: reportId,
    status: report.status
  });

  await logEvent(rootDir, 'quality', 'quality.check.completed', {
    task_id: taskId,
    report_id: reportId,
    status: report.status
  });

  return report;
}

export async function readQualityReport(rootDir, reportId) {
  const paths = reportPaths(rootDir, reportId);
  if (!(await fileExists(paths.json))) {
    return null;
  }
  return readJson(paths.json, null);
}
