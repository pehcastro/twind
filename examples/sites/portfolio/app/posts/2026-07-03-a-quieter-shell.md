# A quieter shell

<Meta date="2026-07-03" tags="shell, habits" />

My prompt used to show the time, the battery, the git branch, the exit code and the weather. I read none of it. This summer I took everything out and added back only what earned its place.

> A prompt is read a thousand times a day. Every character in it is a tax.

## What stayed

The directory, shortened to its last two parts, and the branch, only inside a repository.

```bash
prompt() {
  local dir="${PWD/#$HOME/~}"
  local branch=$(git branch --show-current 2>/dev/null)
  PS1="${dir##*/*/} ${branch:+($branch) }$ "
}
PROMPT_COMMAND=prompt
```

## What I learned

Removing things is harder than adding them, because each one had a reason once. Write the reason down, wait a week, and see if you missed it.
