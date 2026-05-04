import path from 'node:path';
import crypto from 'node:crypto';
import { ensureDir, fileExists, readJson, writeJson, readYaml, writeYaml } from '../lib/fs.js';
import { getWorkspacePaths } from './paths.js';
import { logEvent } from './logging.js';

function ownerPath(rootDir) {
  return path.join(getWorkspacePaths(rootDir).authDir, 'owner.json');
}

function defaultOwner(rootDir, name = 'local-owner') {
  return {
    actor_id: `owner-${crypto
      .createHash('sha1')
      .update(`${rootDir}:${name}`)
      .digest('hex')
      .slice(0, 12)}`,
    display_name: name,
    mode: 'local_single_user',
    created_at: new Date().toISOString()
  };
}

export async function initOwner(rootDir, { name = 'local-owner' } = {}) {
  const paths = getWorkspacePaths(rootDir);
  await ensureDir(paths.authDir);

  const owner = defaultOwner(rootDir, name);
  await writeJson(ownerPath(rootDir), owner);

  const access = (await readYaml(paths.accessYaml, null)) ?? {
    schema_version: 1,
    access: {
      mode: 'local_single_user',
      local_owner_id: owner.actor_id,
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

  access.access = access.access ?? {};
  access.access.mode = 'local_single_user';
  access.access.local_owner_id = owner.actor_id;
  access.access.organization_id = null;
  access.access.project_roles = access.access.project_roles ?? { owner: ['*'] };
  access.access.approval_policy = access.access.approval_policy ?? {};
  access.access.audit = access.access.audit ?? { enabled: true };

  await writeYaml(paths.accessYaml, access);
  await logEvent(rootDir, 'audit', 'auth.owner.initialized', {
    actor_id: owner.actor_id,
    display_name: owner.display_name
  });

  return owner;
}

export async function getCurrentOwner(rootDir) {
  const file = ownerPath(rootDir);
  if (!(await fileExists(file))) {
    return null;
  }
  return readJson(file, null);
}

export async function whoami(rootDir) {
  const owner = await getCurrentOwner(rootDir);
  if (!owner) {
    return {
      actor_id: 'anonymous',
      display_name: 'anonymous',
      mode: 'unknown'
    };
  }
  return owner;
}

export async function createAuthContext(rootDir, action, { actorId = null, risk = 'normal' } = {}) {
  const actor = actorId ? { actor_id: actorId, display_name: actorId } : await whoami(rootDir);
  const isHighRisk = new Set(['git.push', 'git.tag', 'release.publish', 'deploy.run', 'server.write']);
  const decision = isHighRisk.has(action) ? 'REQUIRE_APPROVAL' : 'ALLOW';

  const context = {
    actor_id: actor.actor_id,
    action,
    decision,
    risk,
    created_at: new Date().toISOString()
  };

  await logEvent(rootDir, 'audit', 'auth.context.created', context);
  return context;
}
