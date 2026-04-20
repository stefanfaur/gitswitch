# gitswitch

Per-user git identity + Gitea PAT manager for teammates sharing a unix account.

Each teammate's identity and Personal Access Token live in a password-encrypted
blob under `$XDG_CONFIG_HOME/gitswitch/`. `gitswitch use <name>` launches a
subshell that sets `GIT_AUTHOR_*`, `GIT_COMMITTER_*`, and `GIT_ASKPASS` for git
only — `$HOME` is untouched, so nobody's `~/.gitconfig` or `~/.git-credentials`
is shared or mutated.

## Install

```bash
make build
sudo install -m 0755 gitswitch /usr/local/bin/gitswitch
```

Or:

```bash
go install github.com/stefanfaur/gitswitch/cmd/gitswitch@latest
```

## Quick start

```bash
gitswitch add alice           # name, email, Gitea PAT, password
gitswitch use alice           # subshell with alice's creds
git clone https://gitea.example.com/alice/repo.git
cd repo && git commit -am "..." && git push
exit                          # back to plain shell, creds gone
```

## Shell prompt

Show the active user in your prompt. Add to `~/.bashrc`:

```bash
[ -n "$GITSWITCH_USER" ] && PS1="(git:$GITSWITCH_USER) $PS1"
```

Or `~/.zshrc`:

```zsh
[ -n "$GITSWITCH_USER" ] && PROMPT="(git:$GITSWITCH_USER) $PROMPT"
```

## Commands

| Command | Purpose |
|---|---|
| `add <name>` | Configure new user. Prompts for name, email, PAT, password. |
| `list` | List configured users. |
| `rm <name>` | Remove user. Requires password. |
| `use <name>` | Launch subshell with user's git credentials. |
| `rotate <name>` | Replace PAT, same password. |
| `passwd <name>` | Change password. |
| `whoami` | Print active user or `none`. |

Inside a `use` session, the binary invokes itself as `GIT_ASKPASS` when git
asks for HTTPS credentials. It reads the PAT from env and hands it to git
without writing to disk.

## File layout

```
$XDG_CONFIG_HOME/gitswitch/          # 0700
$XDG_CONFIG_HOME/gitswitch/alice.age # 0600, age scrypt blob
```

If `XDG_CONFIG_HOME` is unset, `~/.config/gitswitch/` is used.

## Threat model

- **Encryption at rest**: blobs use [age](https://age-encryption.org) scrypt
  passphrase recipient, work factor N=2^18. A stolen disk image without the
  password yields no credentials.
- **Runtime exposure**: inside a `use` subshell, `GITSWITCH_PAT` is in the
  process environment and visible to any process the same unix user runs
  (e.g. via `/proc/<pid>/environ` on Linux). This is the same trust boundary
  as the shared account itself — if you need hard isolation between teammates,
  create individual unix accounts.
- **Hardening**: on startup the binary sets `RLIMIT_CORE=0` (no core dumps)
  and on Linux sets `PR_SET_DUMPABLE=0` (no `/proc/self/mem`, no ptrace by
  the same uid). Best-effort; errors are ignored.
- **TOCTOU**: blob reads use `O_NOFOLLOW` and verify regular-file + mode 0600
  + owning uid after `lstat`. Symlink swaps are refused.
- **Signal safety**: tty echo is saved before every password prompt and
  restored on `SIGINT`/`SIGTERM`/`SIGHUP`, so killing `gitswitch` mid-prompt
  never leaves your terminal in a non-echoing state.

## Known incompatibilities

- `user.useConfigOnly=true` in `~/.gitconfig` — git refuses
  `GIT_AUTHOR_NAME`/`GIT_COMMITTER_NAME` from env. Unset that config key.
- Repo-local `commit.gpgsign=true` + `user.signingkey` — v1 does not support
  signing. Disable or sign manually outside the subshell.
- Pre-existing persistent credential helpers with stored Gitea creds
  (`store`, `cache`). Clear them first:
  ```bash
  git credential-store erase <<< 'protocol=https
  host=gitea.example.com'
  ```

## Uninstall

```bash
sudo rm /usr/local/bin/gitswitch
rm -rf "${XDG_CONFIG_HOME:-$HOME/.config}/gitswitch"
```

## Platforms

Linux and macOS. Windows not supported.
