import fs from 'node:fs/promises';
import path from 'node:path';
import { fileExists } from '../lib/fs.js';

export const MOYUAN_DIR_NAME = '.moyuan';

export function getMoyuanDir(rootDir) {
  return path.join(rootDir, MOYUAN_DIR_NAME);
}

export function getWorkspacePaths(rootDir) {
  const moyuanDir = getMoyuanDir(rootDir);
  return {
    rootDir,
    moyuanDir,
    projectYaml: path.join(moyuanDir, 'project.yaml'),
    repositoryYaml: path.join(moyuanDir, 'repository.yaml'),
    accessYaml: path.join(moyuanDir, 'policies', 'access.yaml'),
    permissionsYaml: path.join(moyuanDir, 'policies', 'permissions.yaml'),
    loggingYaml: path.join(moyuanDir, 'policies', 'logging.yaml'),
    projectDir: path.join(moyuanDir),
    authDir: path.join(moyuanDir, 'auth'),
    logsDir: path.join(moyuanDir, 'logs'),
    lifecycleDir: path.join(moyuanDir, 'lifecycle'),
    epicsDir: path.join(moyuanDir, 'lifecycle', 'epics'),
    issuesDir: path.join(moyuanDir, 'lifecycle', 'issues'),
    issueGraphsDir: path.join(moyuanDir, 'lifecycle', 'issue-graphs'),
    schedulesDir: path.join(moyuanDir, 'lifecycle', 'schedules'),
    runsDir: path.join(moyuanDir, 'lifecycle', 'runs'),
    qualityDir: path.join(moyuanDir, 'lifecycle', 'quality'),
    reviewsDir: path.join(moyuanDir, 'lifecycle', 'reviews'),
    releasesDir: path.join(moyuanDir, 'lifecycle', 'releases'),
    deploymentsDir: path.join(moyuanDir, 'lifecycle', 'deployments'),
    comprehensionDir: path.join(moyuanDir, 'comprehension'),
    memoryDir: path.join(moyuanDir, 'memory'),
    agentsDir: path.join(moyuanDir, 'agents'),
    runtimesDir: path.join(moyuanDir, 'runtimes'),
    skillsDir: path.join(moyuanDir, 'skills'),
    tmpDir: path.join(moyuanDir, 'tmp'),
    locksDir: path.join(moyuanDir, '.locks')
  };
}

export async function resolveWorkspaceRoot(startDir = process.cwd()) {
  let current = path.resolve(startDir);

  while (true) {
    if (await fileExists(getMoyuanDir(current))) {
      return current;
    }

    const parent = path.dirname(current);
    if (parent === current) {
      return null;
    }
    current = parent;
  }
}

export async function ensureWorkspaceDirs(rootDir, dirs) {
  for (const dir of dirs) {
    await fs.mkdir(dir, { recursive: true });
  }
}
