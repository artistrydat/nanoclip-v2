<div align="center">

<img src="ui/dist/favicon.svg" width="80" alt="NanoClip logo" />

# NanoClip

**AI agent orchestration that fits in your pocket — runs on Android, offline, free.**

[![Version](https://img.shields.io/badge/version-v0.4.0-6366f1?style=flat-square)](https://github.com/artistrydat/nanoclip-v2/releases)
[![License](https://img.shields.io/badge/license-MIT-22c55e?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Android%20%7C%20Linux%20%7C%20macOS-f59e0b?style=flat-square)](#)
[![Backend](https://img.shields.io/badge/backend-Go%20%2B%20Gin%20%2B%20GORM-00ADD8?style=flat-square)](#)
[![Frontend](https://img.shields.io/badge/frontend-React%20%2B%20Tailwind-38BDF8?style=flat-square)](#)
[![Issues](https://img.shields.io/github/issues/artistrydat/nanoclip-v2?style=flat-square&color=ef4444)](https://github.com/artistrydat/nanoclip-v2/issues)

<br/>

*Create AI agents → assign them issues → they work, you watch.*  
No cloud. No subscription. No monthly bill.

[**Full Termux Guide**](#android--termux-setup-guide) · [**Telegram Plugin**](docs/telegram-plugin.md) · [**Troubleshooting**](#troubleshooting) · [**Report a Bug**](https://github.com/artistrydat/nanoclip-v2/issues)

</div>

---

## What is NanoClip?

NanoClip is an open-source AI agent platform that runs as a **single binary on your Android phone**. You create agents, assign them to tasks (called *issues*), and they respond, plan, and act — all stored locally on your device. No data leaves your phone unless you want it to.

### Supported AI providers

| Provider | How it works |
|---|---|
| **Ollama** (local) | Runs AI models directly on your phone — fully offline |
| **OpenRouter** | Routes to 200+ cloud models with a single API key |
| **HTTP** | Connect any OpenAI-compatible API endpoint |

---

## Features

- **Multi-agent boards** — organize agents into companies and teams
- **Issue tracker** — assign tasks to agents, track progress, leave comments
- **Plugin system** — extend with Telegram, webhooks, and more
- **Approval flows** — agents ask before taking irreversible actions
- **Runs on Android** — single binary, Termux, ARM64 native
- **MariaDB or SQLite** — your choice of database backend

### Telegram Plugin

Get push notifications, inline approve/reject buttons, and two-way issue comments — all inside Telegram. Full setup guide: **[docs/telegram-plugin.md](docs/telegram-plugin.md)**

---

## Quick start (desktop / Linux / macOS)

```bash
git clone https://github.com/artistrydat/nanoclip-v2.git
cd nanoclip-v2
cp .env.example .env        # edit JWT_SECRET at minimum
cd go-server
go build -o nanoclip .
./nanoclip
```

Open `http://localhost:8080` — done.

---

## Android / Termux Setup Guide

> **Who this guide is for:** Anyone who wants to run NanoClip on an Android phone using Termux — even if you have never typed a command before.  
> Every step is explained in plain language. Take it slow, copy each command exactly, and you will have NanoClip running on your phone.

---

### Before you start — check your Android version

Works on **Android 7.0 or newer**. You need at least **2 GB free storage** and an internet connection for the first install.

---

## Part 1 — Install Termux

**Do NOT install Termux from the Google Play Store** — that version is old and broken. Use F-Droid.

### Step 1 — Install F-Droid

1. Open your phone's browser and go to **https://f-droid.org**
2. Tap **"Download F-Droid"** and install the file
3. If asked about *"unknown sources"*, tap **Settings → Allow from this source**, then go back and tap **Install**

### Step 2 — Install Termux from F-Droid

1. Open **F-Droid**, search for `Termux`, and tap **Install**
2. Tap **Open** when done — you will see a black terminal screen, which is normal

### Step 3 — Give Termux storage permission

```
termux-setup-storage
```

Tap **Allow** when the popup appears.  
If no popup appears: go to **Android Settings → Apps → Termux → Permissions → Storage → Allow**.

---

## Part 2 — Install the required tools

### Step 4 — Update Termux

```
pkg update -y && pkg upgrade -y
```

Press **Enter** to accept any prompts. Takes 3–10 minutes.

### Step 5 — Install Go, MariaDB, and Git

```
pkg install -y golang mariadb git
```

Takes 5–15 minutes. Your screen may turn off — the install continues.

### Step 6 — Install pnpm

> **Note:** pnpm is only needed for desktop builds. On Termux you will skip the UI build (it comes pre-built in this repo). This step installs the package manager in case you later want to run the dev server.

```
pkg install -y nodejs-lts
npm install -g pnpm
```

---

## Part 3 — Download NanoClip

### Step 7 — Download the source code

```
git clone https://github.com/artistrydat/nanoclip-v2.git
```

### Step 8 — Enter the NanoClip folder

```
cd nanoclip-v2
```

> Run all future commands from inside this folder.

---

## Part 4 — Set up the database (MariaDB)

> **Note on warnings:** Newer MariaDB versions show *"Deprecated program name, use mariadb instead of mysql"*. These are safe to ignore — the commands still work.

### Step 9 — Initialize MariaDB (first time only)

```
mariadb-install-db
```

> If you see **"mysql.user table already exists"** — skip to Step 10.

### Step 10 — Start the MariaDB database server

```
mariadbd-safe --datadir="$PREFIX/var/lib/mysql" &
```

Wait 5–10 seconds, then press **Enter** for a clean prompt.

> If `mariadbd-safe` is not found, try: `mysqld_safe --datadir="$PREFIX/var/lib/mysql" &`

### Step 11 — Open the database console

```
mariadb -u root
```

You will see `MariaDB [(none)]>` — you are inside the database.

### Step 12 — Create the NanoClip database and user

Run **each line below one at a time**, waiting for `Query OK` before the next:

```sql
CREATE DATABASE IF NOT EXISTS nanoclip CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

```sql
DROP USER IF EXISTS 'nanoclip'@'localhost';
```

```sql
DROP USER IF EXISTS 'nanoclip'@'127.0.0.1';
```

```sql
CREATE USER 'nanoclip'@'localhost' IDENTIFIED BY 'nanoclip123';
```

```sql
CREATE USER 'nanoclip'@'127.0.0.1' IDENTIFIED BY 'nanoclip123';
```

```sql
GRANT ALL PRIVILEGES ON nanoclip.* TO 'nanoclip'@'localhost';
```

```sql
GRANT ALL PRIVILEGES ON nanoclip.* TO 'nanoclip'@'127.0.0.1';
```

```sql
FLUSH PRIVILEGES;
```

```sql
EXIT;
```

> **Why `DROP USER` first?** If you ever ran setup before, the old user entry stays in the database. `IF NOT EXISTS` silently skips the creation and keeps the old broken entry. Dropping first guarantees a clean user with the correct password.

### Step 13 — Test that the database works

```
mariadb -h 127.0.0.1 -P 3306 -u nanoclip -pnanoclip123 nanoclip
```

> Use `-h 127.0.0.1` (not `localhost`). On modern MariaDB, `localhost` uses a UNIX socket which may reject password auth. `127.0.0.1` forces TCP.

If you see `MariaDB [nanoclip]>` — it worked! Type `EXIT;` to return.

---

## Part 5 — Configure NanoClip

### Step 14 — Create the settings file

```
cat > .env << 'ENVEOF'
MARIADB_DSN=nanoclip:nanoclip123@tcp(127.0.0.1:3306)/nanoclip?charset=utf8mb4&parseTime=True&loc=UTC
JWT_SECRET=my-secret-key-change-this-please
GO_PORT=8080
ENVEOF
```

> Replace `my-secret-key-change-this-please` with any random phrase you like (e.g. `blue-sunset-ocean-2025`).

---

## Part 6 — Build NanoClip

> ### 🎉 Termux users — skip Steps 15 and 16!
>
> The web interface is **already pre-built** and included in this repository (`ui/dist/`).  
> The Go server serves these files automatically — you do not need Node.js or pnpm to build anything.
>
> **Jump straight to Step 17** to build only the Go binary.
>
> Steps 15–16 are kept here for developers on desktop machines who want to modify the UI.

---

### Step 15 — Download the web interface dependencies *(desktop only)*

```
pnpm install
```

Takes 5–10 minutes. Wait for `$` before continuing.

---

### Step 16 — Build the web interface *(desktop only)*

On a desktop/laptop:

```
pnpm --filter @nanoclip/ui build
```

On Termux if you specifically need to rebuild the UI (requires a lot of RAM — close all other apps first):

```
NODE_OPTIONS="--max-old-space-size=384" pnpm --filter @nanoclip/ui build:termux
```

---

### Step 17 — Build the NanoClip server

```
cd go-server && go build -o nanoclip-build . && mv nanoclip-build nanoclip && cd ..
```

Nothing will appear for several minutes — that is normal. When `$` appears again, the binary is ready.

If it fails with a memory error:

```
cd go-server && GOFLAGS="-p=1" go build -o nanoclip-build . && mv nanoclip-build nanoclip && cd ..
```

---

## Part 7 — Run NanoClip

### Step 18 — Start the database (every time you open Termux)

```
mariadbd-safe --datadir="$PREFIX/var/lib/mysql" &
```

Wait 5–10 seconds, then press **Enter**.

### Step 19 — Start NanoClip

```
./go-server/nanoclip
```

When you see **"NanoClip listening on 0.0.0.0:8080"**, it is ready.

### Step 20 — Open NanoClip in your browser

Open any browser and go to: **`http://localhost:8080`**

Sign up with any email and password — stored only on your phone.

---

## Part 8 — Keep NanoClip running in the background

### Option A — Keep Termux open (simplest)

Keep the Termux notification visible. Go to **Battery Settings → Termux → Don't optimize**.

### Option B — Run in background with nohup

```
nohup bash -c 'mariadbd-safe --datadir="$PREFIX/var/lib/mysql"; sleep 10; ./go-server/nanoclip' > ~/nanoclip.log 2>&1 &
```

Check logs: `cat ~/nanoclip.log` | Stop: `pkill mariadbd; pkill nanoclip`

### Option C — Auto-start on boot (advanced)

Install **Termux:Boot** from F-Droid, then:

```
mkdir -p ~/.termux/boot
cat > ~/.termux/boot/start-nanoclip.sh << 'BOOTEOF'
#!/data/data/com.termux/files/usr/bin/bash
cd ~/nanoclip-v2
mariadbd-safe --datadir="$PREFIX/var/lib/mysql" &
sleep 12
./go-server/nanoclip >> ~/nanoclip.log 2>&1
BOOTEOF
chmod +x ~/.termux/boot/start-nanoclip.sh
```

---

## Quick reference

| What you want | Command |
|---|---|
| Start the database | `mariadbd-safe --datadir="$PREFIX/var/lib/mysql" &` |
| Start NanoClip | `./go-server/nanoclip` |
| Stop everything | `pkill mariadbd; pkill nanoclip` |
| View live log | `tail -f ~/nanoclip.log` |
| Go into NanoClip folder | `cd ~/nanoclip-v2` |
| Open in browser | `http://localhost:8080` |

---

## Troubleshooting

<details>
<summary><strong>"mysql.user table already exists" during mariadb-install-db</strong></summary>

MariaDB was already initialized. Skip Step 9 and go directly to Step 10.
</details>

<details>
<summary><strong>"Access denied for user 'nanoclip'@'localhost'" in Step 13</strong></summary>

This is a MariaDB 12 socket authentication issue. Always use `-h 127.0.0.1`:

```
mariadb -h 127.0.0.1 -P 3306 -u nanoclip -pnanoclip123 nanoclip
```

If still denied, re-run Step 12 — start with the `DROP USER IF EXISTS` lines to clear the stale entry.
</details>

<details>
<summary><strong>"Out of memory" or "SIGABRT" during Step 16 (UI build)</strong></summary>

The UI build requires ~900MB RAM which most phones cannot spare. This is why Step 16 is skipped for Termux — the pre-built files are already in the repo. If you specifically need to rebuild:

1. Close all other apps
2. Try `NODE_OPTIONS="--max-old-space-size=256" pnpm --filter @nanoclip/ui build:termux`
3. If still failing — use a desktop machine to build and `git push` the dist folder to your fork
</details>

<details>
<summary><strong>"Can't connect to local MySQL server" after starting NanoClip</strong></summary>

MariaDB is not running. Start it:

```
mariadbd-safe --datadir="$PREFIX/var/lib/mysql" &
```

Wait 10 seconds, then restart NanoClip.
</details>

<details>
<summary><strong>"signal: killed" or "out of memory" during Step 17 (Go build)</strong></summary>

```
cd go-server && GOFLAGS="-p=1" go build -o nanoclip-build . && mv nanoclip-build nanoclip && cd ..
```

`-p=1` limits to one CPU core at a time, using much less memory.
</details>

<details>
<summary><strong>"Deprecated program name" warnings everywhere</strong></summary>

Messages like *"Deprecated program name. Use 'mariadb' instead"* are safe to ignore. This guide already uses the new names where possible.
</details>

<details>
<summary><strong>Page does not load in browser</strong></summary>

- Use `http://` not `https://`
- Try `http://127.0.0.1:8080` instead of `localhost`
- Make sure NanoClip is running — you should see "NanoClip listening" in Termux
</details>

---

## Plugins

### Telegram

Get Telegram notifications, approve/reject buttons, and issue replies — directly from your Telegram app.

**→ Full setup guide: [docs/telegram-plugin.md](docs/telegram-plugin.md)**

---

## Updating NanoClip

```bash
cd ~/nanoclip-v2
git pull
cd go-server && go build -o nanoclip-build . && mv nanoclip-build nanoclip && cd ..
```

The UI is pre-built and updated automatically with `git pull` — no rebuild needed.

---

## Configuration reference

Edit `.env`:

| Setting | What it does | Default |
|---|---|---|
| `MARIADB_DSN` | Database connection string | required |
| `GO_PORT` | Port for the web interface | `8080` |
| `JWT_SECRET` | Security key — set to any random phrase | auto-generated |
| `NANOCLIP_DATA_DIR` | SQLite storage path (if not using MariaDB) | `~/.nanoclip/` |

After editing `.env`, restart NanoClip (Ctrl+C, then Step 19).

---

## Contributing

Pull requests are welcome. Please open an issue first to discuss any large changes.

- [Open an issue](https://github.com/artistrydat/nanoclip-v2/issues)
- [Read the Telegram plugin guide](docs/telegram-plugin.md)

---

## License

MIT — free to use, modify, and distribute.

---

<div align="center">

Made for people who want AI agents without the cloud bill.

</div>
