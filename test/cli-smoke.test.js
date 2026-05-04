import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { run } from '../src/cli.js';
import { runCommand } from '../src/lib/process.js';

async function createTempRepo() {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'moyuan-test-'));
  await runCommand('git', ['init', '-q'], { cwd: root });
  await fs.writeFile(
    path.join(root, 'package.json'),
    JSON.stringify(
      {
        type: 'module',
        scripts: {
          test: 'node --test'
        }
      },
      null,
      2
    ) + '\n'
  );
  await runCommand('git', ['add', 'package.json'], { cwd: root });
  await runCommand('git', ['-c', 'user.email=test@example.com', '-c', 'user.name=test', 'commit', '-qm', 'init'], {
    cwd: root
  });
  return root;
}

test('local project add creates workspace, owner, comprehension, graph, and quality report', async () => {
  const root = await createTempRepo();

  const added = await run(['project', 'add', '--local', root, '--root', root], { cwd: root });
  assert.equal(added.exitCode, 0);
  assert.match(added.stdout, /project added:/);

  const whoami = await run(['auth', 'whoami', '--root', root], { cwd: root });
  assert.equal(whoami.exitCode, 0);
  assert.match(whoami.stdout, /local_single_user/);

  const graph = await run(['issue', 'graph', 'phase1-epic', '--root', root], { cwd: root });
  assert.equal(graph.exitCode, 0);
  assert.match(graph.stdout, /phase1-001/);

  const quality = await run(['quality', 'check', 'phase1-001', '--root', root], { cwd: root });
  assert.equal(quality.exitCode, 0);
  const report = JSON.parse(quality.stdout);
  assert.equal(report.status, 'passed');
  assert.equal(report.checks.some((check) => check.type === 'test' && check.command === 'npm test'), true);
});
