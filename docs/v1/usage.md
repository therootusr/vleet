## Usage guide: vleet

### Requirements
- Go **1.24+**

### Build from source

```bash
go test ./...
go build -o vleet ./cmd/vleet
```

### Install

```bash
go install ./cmd/vleet
```

The `vleet` binary will be installed to `$GOBIN` (or `$(go env GOPATH)/bin`). Ensure it’s on your `PATH`.

### Configure (YAML)

Initialize a config file:

```bash
vleet config init
vleet config show
```

Edit the config file to set LeetCode auth cookies (treat these as secrets):

```yaml
editor: vim
default_lang: cpp
leetcode:
  session: "YOUR_LEETCODE_SESSION"
  csrftoken: "YOUR_CSRFTOKEN"
```

How to obtain `LEETCODE_SESSION` and `csrftoken`:
- Log in to LeetCode in your browser (`leetcode.com`).
- Open browser DevTools.
- Go to **Application / Storage** → **Cookies** → `https://leetcode.com`.
- Copy:
  - Cookie **`LEETCODE_SESSION`** → put it in config as `leetcode.session` (_don't embed `LEETCODE_SESSION` in the cookie value; put only the cookie value in the config_)
  - Cookie **`csrftoken`** → put it in config as `leetcode.csrftoken` (_don't embed `csrftoken` in the cookie value_)

CSRF note:
- If `csrftoken` is missing/incorrect, submits can fail with **CSRF verification failed** (often returned as HTTP 403 with an HTML/text error page).

Notes:
- The config file must have permissions **0600** (vleet will refuse insecure perms).
- You can override the config location with `VLEET_CONFIG_PATH=/path/to/config.yaml`.

### Run

Fetch a problem into a workspace in the **current directory**:

```bash
vleet fetch two-sum --lang cpp
```

Flag ordering note:
- vleet currently uses Go’s stdlib `flag` parsing, which means **flags after positional args will not be parsed**.
- You must put flags **before** the problem key:

```bash
vleet fetch --lang cpp two-sum
vleet solve --lang cpp --submit two-sum
vleet submit --lang cpp two-sum
```

Open editor to solve:

```bash
vleet solve two-sum --lang cpp
```

Submit an existing workspace solution:

```bash
vleet submit two-sum --lang cpp
```

Solve and submit after editor exits:

```bash
vleet solve two-sum --lang cpp --submit
```

Additional notes:
- Workspaces are created as `./<titleSlug>/` and solutions as `solution.<ext>` (e.g. `./two-sum/solution.cpp`).
- vleet **does not overwrite** an existing `solution.<ext>` by default.
- Add `--json` to `fetch/solve/submit` for JSON output.
