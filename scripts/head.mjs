import { execFileSync, spawnSync } from "node:child_process";
import { existsSync, mkdirSync, readdirSync, renameSync, rmSync } from "node:fs";
import { basename, join } from "node:path";

const root = process.cwd();
const head = join(root, ".twind", "head");
const cache = join(root, ".twind", "cache", "run");
const git = (...args) => execFileSync("git", args, { stdio: ["ignore", "pipe", "inherit"] }).toString().trim();
const describe = (dir, rev) => git("-C", dir, "log", "-1", "--format=%H%n%h %s", rev).split("\n");
const announce = ([, label]) => console.error(`[twind] running at ${label}`);

function checkout() {
  if (!existsSync(join(head, ".git"))) {
    git("clone", "--quiet", "--no-hardlinks", root, head);
  }
  git("-C", head, "fetch", "--quiet", "origin");
  git("-C", head, "checkout", "--quiet", "--force", "--detach", "origin/main");
  return describe(head, "HEAD");
}

function binary(pkg) {
  const name = basename(pkg);
  const suffix = process.platform === "win32" ? ".exe" : "";
  let at = describe(root, "main");
  let exe = join(cache, `${name}-${at[0]}${suffix}`);
  if (!existsSync(exe)) {
    at = checkout();
    exe = join(cache, `${name}-${at[0]}${suffix}`);
    const partial = `${exe}.partial${suffix}`;
    mkdirSync(cache, { recursive: true });
    const build = spawnSync("go", ["build", "-o", partial, pkg], { cwd: head, stdio: "inherit" });
    if (build.status !== 0) {
      rmSync(partial, { force: true });
      process.exit(build.status ?? 1);
    }
    renameSync(partial, exe);
    for (const old of readdirSync(cache).filter((f) => f.startsWith(`${name}-`) && join(cache, f) !== exe)) {
      try {
        rmSync(join(cache, old));
      } catch {}
    }
  }
  announce(at);
  return exe;
}

const [mode, ...rest] = process.argv.slice(2);
let run;
if (mode === "run") {
  const [pkg, ...args] = rest;
  run = spawnSync(binary(pkg), args, { cwd: head, stdio: "inherit" });
} else if (mode === "window") {
  const [size, title, pkg, ...args] = rest;
  const wt = ["-w", "new", "--size", size, "-d", head, "--title", title, "--suppressApplicationTitle", "--", binary(pkg), ...args];
  run = spawnSync("wt", wt, { stdio: "inherit" });
} else {
  announce(checkout());
  run = spawnSync(mode, rest, { cwd: head, stdio: "inherit", shell: process.platform === "win32" && mode !== "go" });
}
process.exit(run.status ?? 1);
