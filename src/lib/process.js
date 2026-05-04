import { spawn } from 'node:child_process';

function collectStream(stream) {
  return new Promise((resolve) => {
    let data = '';
    stream.on('data', (chunk) => {
      data += chunk.toString('utf8');
    });
    stream.on('end', () => resolve(data));
  });
}

export async function runCommand(command, args = [], options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      cwd: options.cwd,
      env: options.env,
      stdio: ['ignore', 'pipe', 'pipe'],
      shell: false
    });

    const stdoutPromise = collectStream(child.stdout);
    const stderrPromise = collectStream(child.stderr);

    child.on('error', reject);
    child.on('close', async (code) => {
      const stdout = await stdoutPromise;
      const stderr = await stderrPromise;
      resolve({ code, stdout, stderr });
    });
  });
}

export async function runShell(commandLine, options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(commandLine, {
      cwd: options.cwd,
      env: options.env,
      stdio: ['ignore', 'pipe', 'pipe'],
      shell: true
    });

    const stdoutPromise = collectStream(child.stdout);
    const stderrPromise = collectStream(child.stderr);

    child.on('error', reject);
    child.on('close', async (code) => {
      const stdout = await stdoutPromise;
      const stderr = await stderrPromise;
      resolve({ code, stdout, stderr });
    });
  });
}
