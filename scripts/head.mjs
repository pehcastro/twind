import { execFileSync, spawnSync } from "node:child_process";
import { existsSync } from "node:fs";
import { join } from "node:path";

const root = process.cwd();
const head = join(root, ".twind", "head");
const git = (...args) => execFileSync("git", args, { stdio: ["ignore", "pipe", "inherit"] }).toString().trim();

if (!existsSync(join(head, ".git"))) {
  git("clone", "--quiet", "--no-hardlinks", root, head);
}
git("-C", head, "fetch", "--quiet", "origin");
git("-C", head, "checkout", "--quiet", "--force", "--detach", "origin/main");

const [cmd, ...args] = process.argv.slice(2);
console.error(`[twind] running at ${git("-C", head, "log", "-1", "--format=%h %s")}`);
const run = spawnSync(cmd, args, { cwd: head, stdio: "inherit", shell: process.platform === "win32" && cmd !== "go" });
process.exit(run.status ?? 1);
