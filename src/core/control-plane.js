import path from 'node:path';
import { fileExists, readJson, writeJson, ensureDir } from '../lib/fs.js';
import { getWorkspacePaths } from './paths.js';

export function getControlPlanePaths(rootDir = process.cwd()) {
  const paths = getWorkspacePaths(rootDir);
  return {
    ...paths,
    projectsIndex: path.join(paths.moyuanDir, 'projects.json'),
    projectsDir: path.join(paths.moyuanDir, 'projects')
  };
}

export async function loadProjectRegistry(rootDir = process.cwd()) {
  const { projectsIndex } = getControlPlanePaths(rootDir);
  if (!(await fileExists(projectsIndex))) {
    return {
      schema_version: 1,
      projects: []
    };
  }

  return readJson(projectsIndex, {
    schema_version: 1,
    projects: []
  });
}

export async function saveProjectRegistry(rootDir = process.cwd(), registry) {
  const { projectsIndex } = getControlPlanePaths(rootDir);
  await ensureDir(path.dirname(projectsIndex));
  await writeJson(projectsIndex, registry);
  return registry;
}

export async function registerProject(rootDir = process.cwd(), projectRecord) {
  const registry = await loadProjectRegistry(rootDir);
  const normalized = {
    ...projectRecord,
    registered_at: new Date().toISOString()
  };

  const remaining = registry.projects.filter((item) => item.root !== normalized.root);
  registry.projects = [normalized, ...remaining];
  await saveProjectRegistry(rootDir, registry);
  return normalized;
}

export async function listRegisteredProjects(rootDir = process.cwd()) {
  const registry = await loadProjectRegistry(rootDir);
  return registry.projects ?? [];
}
