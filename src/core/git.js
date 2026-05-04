import path from 'node:path';
import { fileExists, readYaml, writeYaml } from '../lib/fs.js';
import { runCommand } from '../lib/process.js';
import { getWorkspacePaths } from './paths.js';
import { logEvent } from './logging.js';

export async function isGitRepo(rootDir) {
  const result = await runCommand('git', ['rev-parse', '--is-inside-work-tree'], { cwd: rootDir });
  return result.code === 0 && result.stdout.trim() === 'true';
}

export async function getGitStatus(rootDir) {
  if (!(await isGitRepo(rootDir))) {
    return {
      isRepo: false,
      dirty: false,
      branch: null,
      remote: null,
      ahead: null,
      behind: null,
      files: []
    };
  }

  const branch = await runCommand('git', ['branch', '--show-current'], { cwd: rootDir });
  const remote = await runCommand('git', ['remote', 'get-url', 'origin'], { cwd: rootDir });
  const status = await runCommand('git', ['status', '--short'], { cwd: rootDir });

  return {
    isRepo: true,
    dirty: status.stdout.trim().length > 0,
    branch: branch.stdout.trim() || null,
    remote: remote.code === 0 ? remote.stdout.trim() : null,
    ahead: null,
    behind: null,
    files: status.stdout
      .split('\n')
      .map((line) => line.trim())
      .filter(Boolean)
  };
}

export async function listBranches(rootDir) {
  if (!(await isGitRepo(rootDir))) {
    return [];
  }

  const result = await runCommand('git', ['branch', '--format', '%(refname:short)'], { cwd: rootDir });
  if (result.code !== 0) {
    return [];
  }
  return result.stdout
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean);
}

export async function currentBranch(rootDir) {
  const result = await runCommand('git', ['branch', '--show-current'], { cwd: rootDir });
  return result.code === 0 ? result.stdout.trim() || null : null;
}

export async function defaultBranch(rootDir) {
  const remote = await runCommand('git', ['symbolic-ref', 'refs/remotes/origin/HEAD'], { cwd: rootDir });
  if (remote.code === 0) {
    return path.basename(remote.stdout.trim());
  }
  return null;
}

export async function cloneRepository(url, destDir) {
  const result = await runCommand('git', ['clone', url, destDir]);
  if (result.code !== 0) {
    throw new Error(result.stderr.trim() || `git clone failed for ${url}`);
  }
  return destDir;
}

export async function bindLocalRepository(rootDir, repositoryConfig) {
  const paths = getWorkspacePaths(rootDir);
  const next = {
    ...(repositoryConfig ?? {}),
    schema_version: 1,
    repository: {
      ...(repositoryConfig?.repository ?? {}),
      source: {
        ...(repositoryConfig?.repository?.source ?? {}),
        type: 'local_path',
        provider: 'local',
        local_path: rootDir,
        url: null,
        clone_path: null
      }
    }
  };

  await writeYaml(paths.repositoryYaml, next);
  await logEvent(rootDir, 'git', 'repository.bound.local', { rootDir });
  return next;
}

export async function registerRemoteRepository(rootDir, repositoryConfig, url, provider = 'generic_git') {
  const paths = getWorkspacePaths(rootDir);
  const next = {
    ...(repositoryConfig ?? {}),
    schema_version: 1,
    repository: {
      ...(repositoryConfig?.repository ?? {}),
      source: {
        ...(repositoryConfig?.repository?.source ?? {}),
        type: 'remote_git',
        provider,
        url,
        local_path: null
      }
    }
  };

  await writeYaml(paths.repositoryYaml, next);
  await logEvent(rootDir, 'git', 'repository.bound.remote', { url, provider });
  return next;
}

export async function syncRepository(rootDir) {
  if (!(await isGitRepo(rootDir))) {
    throw new Error(`Not a git repository: ${rootDir}`);
  }

  const fetch = await runCommand('git', ['fetch', '--all', '--prune'], { cwd: rootDir });
  if (fetch.code !== 0) {
    throw new Error(fetch.stderr.trim() || 'git fetch failed');
  }

  return {
    branch: await currentBranch(rootDir),
    defaultBranch: await defaultBranch(rootDir)
  };
}

export async function createBranch(rootDir, branchName, base = null) {
  const args = ['checkout', '-b', branchName];
  if (base) {
    args.push(base);
  }
  const result = await runCommand('git', args, { cwd: rootDir });
  if (result.code !== 0) {
    throw new Error(result.stderr.trim() || `git branch create failed: ${branchName}`);
  }
  await logEvent(rootDir, 'git', 'branch.created', { branchName, base });
  return branchName;
}

export async function gitStatusText(rootDir) {
  if (!(await isGitRepo(rootDir))) {
    return 'not a git repository';
  }
  const result = await runCommand('git', ['status', '--short', '--branch'], { cwd: rootDir });
  return result.stdout.trim();
}

export async function ensureRemoteConfigured(rootDir, remote = 'origin') {
  if (!(await fileExists(rootDir))) {
    throw new Error(`Path does not exist: ${rootDir}`);
  }
  const result = await runCommand('git', ['remote', 'get-url', remote], { cwd: rootDir });
  return result.code === 0;
}
