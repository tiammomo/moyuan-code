import path from 'node:path';
import { fileExists } from './lib/fs.js';
import { slugify } from './lib/text.js';
import { ensureWorkspace, loadWorkspace } from './core/workspace.js';
import { initOwner, whoami, createAuthContext } from './core/auth.js';
import { resolveWorkspaceRoot } from './core/paths.js';
import { getGitStatus, listBranches, syncRepository, cloneRepository, bindLocalRepository, registerRemoteRepository } from './core/git.js';
import { logEvent, tailEvents } from './core/logging.js';
import { fullComprehension, incrementalComprehension } from './core/comprehension.js';
import { generatePhase1IssueGraph, loadIssueGraph, loadSchedule } from './core/issues.js';
import { runQualityCheck, readQualityReport } from './core/quality.js';
import { createRun } from './core/run.js';
import { registerProject, listRegisteredProjects } from './core/control-plane.js';

function usage() {
  return [
    'moyuan project add --local <path>',
    'moyuan project add --remote <git-url>',
    'moyuan project list',
    'moyuan auth init-owner [--name <name>]',
    'moyuan auth whoami',
    'moyuan init <path>',
    'moyuan comprehend [--full] [--since <commit>]',
    'moyuan status',
    'moyuan workspace doctor',
    'moyuan git status',
    'moyuan git branch list',
    'moyuan git sync [--comprehend]',
    'moyuan issue graph <epic-id>',
    'moyuan issue schedule <epic-id>',
    'moyuan run <task-id>',
    'moyuan quality check <task-id>',
    'moyuan quality report <report-id>',
    'moyuan logs tail [--stream run] [--limit 20]',
    'moyuan logs query --stream <stream> [--limit 20]'
  ].join('\n');
}

function parseArgs(argv) {
  const [command = null, ...rest] = argv;
  return { command, args: rest };
}

function resolveRootFlag(args, cwd) {
  const root = getFlag(args, '--root', null);
  return root ? path.resolve(cwd, root) : cwd;
}

function getFlag(args, name, fallback = null) {
  const index = args.indexOf(name);
  if (index === -1) return fallback;
  return args[index + 1] ?? fallback;
}

function hasFlag(args, name) {
  return args.includes(name);
}

async function ensureProjectContext(cwd = process.cwd()) {
  const workspaceRoot = await resolveWorkspaceRoot(cwd);
  if (!workspaceRoot) {
    throw new Error('No .moyuan workspace found. Run `moyuan project add --local <path>` or `moyuan init <path>` first.');
  }
  return workspaceRoot;
}

async function ensureControlPlane(cwd = process.cwd()) {
  await ensureWorkspace(cwd);
  const paths = getControlPlanePaths(cwd);
  return paths;
}

async function handleProjectCommand(args, cwd) {
  const sub = args[0];
  if (sub === 'add') {
    const local = getFlag(args, '--local', null);
    const remote = getFlag(args, '--remote', null);
    const dest = getFlag(args, '--dest', null);

    if (local) {
      const rootDir = path.resolve(cwd, local);
      await ensureWorkspace(rootDir);
      const owner = await initOwner(rootDir, { name: path.basename(rootDir) });
      const ws = await loadWorkspace(rootDir);
      await bindLocalRepository(rootDir, ws.repository);
      await fullComprehension(rootDir);
      await registerProject(cwd, {
        id: slugify(path.basename(rootDir)),
        name: path.basename(rootDir),
        root: rootDir,
        source: { type: 'local_path', provider: 'local', path: rootDir },
        owner_id: owner.actor_id,
        status: 'active'
      });
      return { stdout: `project added: ${rootDir}\n`, stderr: '', exitCode: 0 };
    }

    if (remote) {
      const destDir = dest ? path.resolve(cwd, dest) : path.resolve(cwd, '.moyuan', 'projects', slugify(remote));
      await cloneRepository(remote, destDir);
      await ensureWorkspace(destDir);
      const owner = await initOwner(destDir, { name: path.basename(destDir) });
      const ws = await loadWorkspace(destDir);
      await registerRemoteRepository(destDir, ws.repository, remote, 'generic_git');
      await fullComprehension(destDir);
      await registerProject(cwd, {
        id: slugify(path.basename(destDir)),
        name: path.basename(destDir),
        root: destDir,
        source: { type: 'remote_git', provider: 'generic_git', url: remote, clone_path: destDir },
        owner_id: owner.actor_id,
        status: 'active'
      });
      return { stdout: `project added: ${destDir}\n`, stderr: '', exitCode: 0 };
    }

    return { stdout: `missing --local or --remote\n`, stderr: '', exitCode: 1 };
  }

  if (sub === 'list') {
    const projects = await listRegisteredProjects(cwd);
    const lines = projects.map((project) => `- ${project.id} ${project.root} ${project.status ?? 'unknown'}`);
    return { stdout: `${lines.join('\n')}${lines.length ? '\n' : ''}`, stderr: '', exitCode: 0 };
  }

  return { stdout: `unknown project command\n`, stderr: '', exitCode: 1 };
}

async function handleAuthCommand(args, cwd) {
  const sub = args[0];
  const workspaceRoot = await ensureProjectContext(cwd);

  if (sub === 'init-owner') {
    const name = getFlag(args, '--name', path.basename(workspaceRoot));
    const owner = await initOwner(workspaceRoot, { name });
    return { stdout: `${owner.actor_id}\n`, stderr: '', exitCode: 0 };
  }

  if (sub === 'whoami') {
    const current = await whoami(workspaceRoot);
    return { stdout: `${JSON.stringify(current, null, 2)}\n`, stderr: '', exitCode: 0 };
  }

  return { stdout: `unknown auth command\n`, stderr: '', exitCode: 1 };
}

async function handleComprehendCommand(args, cwd) {
  const workspaceRoot = await ensureProjectContext(cwd);
  const since = getFlag(args, '--since', null);
  const profile = await fullComprehension(workspaceRoot, { since: hasFlag(args, '--full') ? null : since });
  return { stdout: `${JSON.stringify(profile, null, 2)}\n`, stderr: '', exitCode: 0 };
}

async function handleWorkspaceCommand(args, cwd) {
  const sub = args[0];
  const workspaceRoot = await ensureProjectContext(cwd);
  const ws = await loadWorkspace(workspaceRoot);
  if (sub === 'doctor') {
    return {
      stdout: `${JSON.stringify({ root: workspaceRoot, project: ws.project, repository: ws.repository, access: ws.access }, null, 2)}\n`,
      stderr: '',
      exitCode: 0
    };
  }
  return { stdout: `unknown workspace command\n`, stderr: '', exitCode: 1 };
}

async function handleGitCommand(args, cwd) {
  const workspaceRoot = await ensureProjectContext(cwd);
  const sub = args[0];
  if (sub === 'status') {
    const status = await getGitStatus(workspaceRoot);
    return { stdout: `${JSON.stringify(status, null, 2)}\n`, stderr: '', exitCode: 0 };
  }
  if (sub === 'branch' && args[1] === 'list') {
    const branches = await listBranches(workspaceRoot);
    return { stdout: `${branches.map((branch) => `- ${branch}`).join('\n')}${branches.length ? '\n' : ''}`, stderr: '', exitCode: 0 };
  }
  if (sub === 'sync') {
    const sync = await syncRepository(workspaceRoot);
    if (hasFlag(args, '--comprehend')) {
      await incrementalComprehension(workspaceRoot, 'git-sync');
    }
    return { stdout: `${JSON.stringify(sync, null, 2)}\n`, stderr: '', exitCode: 0 };
  }
  return { stdout: `unknown git command\n`, stderr: '', exitCode: 1 };
}

async function handleIssueCommand(args, cwd) {
  const workspaceRoot = await ensureProjectContext(cwd);
  const sub = args[0];
  const epicId = args[1] ?? 'phase1-epic';
  if (sub === 'graph') {
    const graph = await loadIssueGraph(workspaceRoot, epicId);
    return { stdout: `${JSON.stringify(graph ?? {}, null, 2)}\n`, stderr: '', exitCode: 0 };
  }
  if (sub === 'schedule') {
    const schedule = await loadSchedule(workspaceRoot, epicId);
    return { stdout: `${JSON.stringify(schedule ?? {}, null, 2)}\n`, stderr: '', exitCode: 0 };
  }
  return { stdout: `unknown issue command\n`, stderr: '', exitCode: 1 };
}

async function handleQualityCommand(args, cwd) {
  const workspaceRoot = await ensureProjectContext(cwd);
  const sub = args[0];
  if (sub === 'check') {
    const taskId = args[1] ?? 'task-unknown';
    const report = await runQualityCheck(workspaceRoot, taskId);
    return { stdout: `${JSON.stringify(report, null, 2)}\n`, stderr: '', exitCode: report.status === 'passed' ? 0 : 1 };
  }
  if (sub === 'report') {
    const reportId = args[1];
    const report = await readQualityReport(workspaceRoot, reportId);
    return { stdout: `${JSON.stringify(report ?? {}, null, 2)}\n`, stderr: '', exitCode: report ? 0 : 1 };
  }
  return { stdout: `unknown quality command\n`, stderr: '', exitCode: 1 };
}

async function handleRunCommand(args, cwd) {
  const workspaceRoot = await ensureProjectContext(cwd);
  const taskId = args[0] ?? 'task-unknown';
  const run = await createRun(workspaceRoot, taskId, {
    issue_id: taskId,
    mode: 'queued'
  });
  return { stdout: `${JSON.stringify(run, null, 2)}\n`, stderr: '', exitCode: 0 };
}

async function handleLogsCommand(args, cwd) {
  const workspaceRoot = await ensureProjectContext(cwd);
  const sub = args[0];
  if (sub === 'tail') {
    const stream = getFlag(args, '--stream', 'run');
    const limit = Number(getFlag(args, '--limit', '20'));
    const events = await tailEvents(workspaceRoot, stream, Number.isFinite(limit) ? limit : 20);
    return { stdout: `${events.join('\n')}${events.length ? '\n' : ''}`, stderr: '', exitCode: 0 };
  }
  return { stdout: `unknown logs command\n`, stderr: '', exitCode: 1 };
}

async function handleProjectRootInit(args, cwd) {
  const target = path.resolve(cwd, args[0] ?? '.');
  await ensureWorkspace(target);
  await initOwner(target, { name: path.basename(target) });
  const profile = await fullComprehension(target);
  await registerProject(cwd, {
    id: slugify(path.basename(target)),
    name: path.basename(target),
    root: target,
    source: { type: 'local_path', provider: 'local', path: target },
    status: 'active'
  });
  return { stdout: `${JSON.stringify(profile, null, 2)}\n`, stderr: '', exitCode: 0 };
}

async function handleStatusCommand(cwd) {
  const workspaceRoot = await ensureProjectContext(cwd);
  const ws = await loadWorkspace(workspaceRoot);
  const gitStatus = await getGitStatus(workspaceRoot);
  const current = await whoami(workspaceRoot);
  return {
    stdout: `${JSON.stringify({ workspace: ws, git: gitStatus, actor: current }, null, 2)}\n`,
    stderr: '',
    exitCode: 0
  };
}

export async function run(argv, { cwd = process.cwd() } = {}) {
  const { command, args } = parseArgs(argv);
  const effectiveCwd = resolveRootFlag(args, cwd);

  try {
    if (!command) {
      return { stdout: `${usage()}\n`, stderr: '', exitCode: 0 };
    }

    if (command === 'project') return await handleProjectCommand(args, effectiveCwd);
    if (command === 'auth') return await handleAuthCommand(args, effectiveCwd);
    if (command === 'init') return await handleProjectRootInit(args, effectiveCwd);
    if (command === 'comprehend') return await handleComprehendCommand(args, effectiveCwd);
    if (command === 'workspace') return await handleWorkspaceCommand(args, effectiveCwd);
    if (command === 'git') return await handleGitCommand(args, effectiveCwd);
    if (command === 'issue') return await handleIssueCommand(args, effectiveCwd);
    if (command === 'quality') return await handleQualityCommand(args, effectiveCwd);
    if (command === 'run') return await handleRunCommand(args, effectiveCwd);
    if (command === 'status') return await handleStatusCommand(effectiveCwd);
    if (command === 'logs') return await handleLogsCommand(args, effectiveCwd);

    return { stdout: `unknown command: ${command}\n\n${usage()}\n`, stderr: '', exitCode: 1 };
  } catch (error) {
    return {
      stdout: '',
      stderr: `${error?.stack ?? error?.message ?? String(error)}\n`,
      exitCode: 1
    };
  }
}
