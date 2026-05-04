import fs from 'node:fs/promises';
import path from 'node:path';
import YAML from 'yaml';

export async function ensureDir(dirPath) {
  await fs.mkdir(dirPath, { recursive: true });
}

export async function fileExists(filePath) {
  try {
    await fs.access(filePath);
    return true;
  } catch {
    return false;
  }
}

export async function readText(filePath, fallback = null) {
  try {
    return await fs.readFile(filePath, 'utf8');
  } catch (error) {
    if (error && error.code === 'ENOENT') {
      return fallback;
    }
    throw error;
  }
}

export async function writeText(filePath, text) {
  await ensureDir(path.dirname(filePath));
  await fs.writeFile(filePath, text, 'utf8');
}

export async function appendText(filePath, text) {
  await ensureDir(path.dirname(filePath));
  await fs.appendFile(filePath, text, 'utf8');
}

export async function readJson(filePath, fallback = null) {
  const text = await readText(filePath, null);
  if (text === null) {
    return fallback;
  }
  return JSON.parse(text);
}

export async function writeJson(filePath, value) {
  await writeText(filePath, `${JSON.stringify(value, null, 2)}\n`);
}

export async function readYaml(filePath, fallback = null) {
  const text = await readText(filePath, null);
  if (text === null) {
    return fallback;
  }
  const parsed = YAML.parse(text);
  return parsed ?? fallback;
}

export async function writeYaml(filePath, value) {
  await writeText(filePath, `${YAML.stringify(value)}\n`);
}

export async function appendJsonl(filePath, value) {
  await appendText(filePath, `${JSON.stringify(value)}\n`);
}

export async function readJsonl(filePath) {
  const text = await readText(filePath, '');
  if (!text) {
    return [];
  }
  return text
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => JSON.parse(line));
}

export async function tailLines(filePath, limit = 20) {
  const text = await readText(filePath, '');
  if (!text) {
    return [];
  }
  return text
    .split('\n')
    .filter(Boolean)
    .slice(-limit);
}
