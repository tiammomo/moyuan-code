import path from 'node:path';
import fs from 'node:fs/promises';
import { ensureDir, fileExists, readYaml, writeYaml } from '../lib/fs.js';
import { slugify } from '../lib/text.js';
import { getWorkspacePaths, ensureWorkspaceDirs } from './paths.js';

const DEFAULT_WORKSPACE_DIRS = [
  'auth',
  'agents',
  'comprehension',
  path.join('lifecycle', 'epics'),
  path.join('lifecycle', 'issues'),
  path.join('lifecycle', 'issue-graphs'),
  path.join('lifecycle', 'schedules'),
  path.join('lifecycle', 'runs'),
  path.join('lifecycle', 'quality'),
  path.join('lifecycle', 'reviews'),
  path.join('lifecycle', 'releases'),
  path.join('lifecycle', 'deployments'),
  'logs',
  'memory',
  'models',
  'policies',
  'runtimes',
  'skills',
  'tmp',
  '.locks'
];

function defaultProjectConfig(rootDir) {
  const baseName = path.basename(rootDir);
  const projectId = slugify(baseName);

  return {
    schema_version: 1,
    project: {
      id: projectId,
      name: baseName,
      root: '.',
      type: 'single-repo',
      description: null
    },
    stack: {
      languages: [],
      frameworks: [],
      package_managers: [],
      build_commands: [],
      test_commands: [],
      lint_commands: []
    },
    workspace: {
      protected_paths: ['.env', '.env.*'],
      writable_paths: ['docs', 'scripts', 'src']
    }
  };
}

function defaultRepositoryConfig(rootDir) {
  return {
    schema_version: 1,
    repository: {
      source: {
        type: 'local_path',
        provider: 'local',
        local_path: rootDir,
        url: null,
        clone_path: null
      },
      default_remote: 'origin',
      default_branch: null
    },
    git: {
      branch_policy: {
        mode: 'task_branch',
        naming: 'moyuan/{issue_id}-{slug}',
        base: 'default_branch',
        sync_before_run: true,
        require_clean_worktree: true,
        allow_auto_commit: false,
        allow_auto_push: false,
        allow_auto_pr: false
      },
      commit_policy: {
        enabled: true,
        format: 'conventional_commits',
        require_issue_ref: true,
        require_quality_ref: true
      }
    }
  };
}

function defaultAccessConfig() {
  return {
    schema_version: 1,
    access: {
      mode: 'local_single_user',
      local_owner_id: null,
      organization_id: null,
      project_roles: {
        owner: ['*']
      },
      approval_policy: {},
      audit: {
        enabled: true
      }
    }
  };
}

function defaultPermissionsConfig() {
  return {
    schema_version: 1,
    permissions: {
      filesystem: {
        writable_paths: ['docs', 'scripts', 'src'],
        protected_paths: ['.env', '.env.*']
      },
      commands: {
        allow: [],
        require_approval: ['git push', 'git tag', 'release publish', 'deploy run'],
        deny: []
      },
      network: {
        enabled: false
      }
    }
  };
}

function defaultLoggingConfig() {
  return {
    schema_version: 1,
    logging: {
      enabled: true,
      format: 'jsonl',
      storage: {
        base_dir: '.moyuan/logs'
      },
      streams: ['run', 'audit', 'error']
    }
  };
}

export function getWorkspaceStateFile(rootDir) {
  return path.join(rootDir, '.moyuan', 'workspace.json');
}

export async function ensureWorkspace(rootDir, options = {}) {
  const paths = getWorkspacePaths(rootDir);

  await ensureWorkspaceDirs(rootDir, [
    paths.moyuanDir,
    ...DEFAULT_WORKSPACE_DIRS.map((dir) => path.join(paths.moyuanDir, dir))
  ]);

  if (!(await fileExists(paths.projectYaml))) {
    await writeYaml(paths.projectYaml, options.projectConfig ?? defaultProjectConfig(rootDir));
  }

  if (!(await fileExists(paths.repositoryYaml))) {
    const repoConfig = options.repositoryConfig ?? defaultRepositoryConfig(rootDir);
    await writeYaml(paths.repositoryYaml, repoConfig);
  }

  if (!(await fileExists(paths.accessYaml))) {
    await ensureDir(path.dirname(paths.accessYaml));
    await writeYaml(paths.accessYaml, options.accessConfig ?? defaultAccessConfig());
  }

  if (!(await fileExists(paths.permissionsYaml))) {
    await writeYaml(paths.permissionsYaml, options.permissionsConfig ?? defaultPermissionsConfig());
  }

  if (!(await fileExists(paths.loggingYaml))) {
    await writeYaml(paths.loggingYaml, options.loggingConfig ?? defaultLoggingConfig());
  }

  const stateFile = getWorkspaceStateFile(rootDir);
  if (!(await fileExists(stateFile))) {
    await fs.writeFile(
      stateFile,
      `${JSON.stringify(
        {
          rootDir: path.resolve(rootDir),
          createdAt: new Date().toISOString()
        },
        null,
        2
      )}\n`,
      'utf8'
    );
  }

  return getWorkspacePaths(rootDir);
}

export async function loadWorkspace(rootDir) {
  const paths = getWorkspacePaths(rootDir);
  return {
    paths,
    project: await readYaml(paths.projectYaml, null),
    repository: await readYaml(paths.repositoryYaml, null),
    access: await readYaml(paths.accessYaml, null),
    permissions: await readYaml(paths.permissionsYaml, null),
    logging: await readYaml(paths.loggingYaml, null)
  };
}

export async function updateWorkspaceProject(rootDir, patch) {
  const paths = getWorkspacePaths(rootDir);
  const current = (await readYaml(paths.projectYaml, null)) ?? defaultProjectConfig(rootDir);
  const next = {
    ...current,
    ...patch,
    project: {
      ...(current.project ?? {}),
      ...(patch.project ?? {})
    },
    stack: {
      ...(current.stack ?? {}),
      ...(patch.stack ?? {})
    },
    workspace: {
      ...(current.workspace ?? {}),
      ...(patch.workspace ?? {})
    }
  };

  await writeYaml(paths.projectYaml, next);
  return next;
}

export async function updateWorkspaceRepository(rootDir, patch) {
  const paths = getWorkspacePaths(rootDir);
  const current = (await readYaml(paths.repositoryYaml, null)) ?? defaultRepositoryConfig(rootDir);
  const next = {
    ...current,
    ...patch,
    repository: {
      ...(current.repository ?? {}),
      ...(patch.repository ?? {})
    },
    git: {
      ...(current.git ?? {}),
      ...(patch.git ?? {})
    }
  };

  await writeYaml(paths.repositoryYaml, next);
  return next;
}
