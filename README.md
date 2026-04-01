# NanoClip — Android / Termux Setup Guide

> **Who this guide is for:** Anyone who wants to run NanoClip on an Android phone using Termux — even if you have never used a terminal or typed a command before.  
> Every step is explained in plain language. Take it slow, copy each command exactly as shown, and you will have NanoClip running on your phone.

---

## What is NanoClip?

NanoClip is an AI agent platform that runs entirely on your Android phone. You create AI agents, assign them to tasks (called *issues*), and they respond, track progress, and work automatically — all without a cloud subscription. Everything runs locally on your device.

---

## Before you start — check your Android version

This guide works on **Android 7 or newer**. If your phone is very old (2013 or earlier), the build step will take a very long time — plug it in and be patient.

**Minimum requirements:**
- Android 7.0 (Nougat) or newer
- At least **2 GB free storage**
- A working internet connection for the first install

---

## Part 1 — Install Termux

Termux is a free terminal app for Android. **Do NOT install it from the Google Play Store** — that version is old and broken. Use F-Droid instead.

### Step 1 — Install F-Droid

1. Open your phone's browser (Chrome, Firefox, etc.)
2. Go to: **https://f-droid.org**
3. Tap the big **"Download F-Droid"** button
4. When the download finishes, tap the file to install it
5. If your phone asks *"Allow install from unknown sources"*, tap **Settings**, turn on **"Allow from this source"**, then go back and tap **Install**
6. After it installs, tap **Open**

---

### Step 2 — Install Termux from F-Droid

1. Open the **F-Droid** app
2. Tap the search icon at the bottom
3. Type: `Termux`
4. Tap the **"Termux"** result and tap **Install**
5. Tap **Open** when done

You will see a black screen with some text — that is completely normal.

---

### Step 3 — Give Termux storage permission

Type the following and press **Enter**:

```
termux-setup-storage
```

Tap **Allow** when the permission popup appears.

> If no popup appears: go to **Android Settings → Apps → Termux → Permissions → Storage → Allow**.

---

## Part 2 — Install the required tools

### Step 4 — Update Termux

```
pkg update -y && pkg upgrade -y
```

> This may ask questions — just press **Enter** to accept defaults. Takes 3–10 minutes.

---

### Step 5 — Install Go, Node.js, MariaDB, and Git

```
pkg install -y golang nodejs-lts mariadb git
```

> Takes 5–15 minutes. Your phone screen may turn off — the install continues in the background.

---

### Step 6 — Install pnpm

```
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

> You must run all future commands from inside this folder.

---

## Part 4 — Set up the database (MariaDB)

> **Note on warning messages:** Newer versions of MariaDB show messages like *"Deprecated program name, use mariadb instead of mysql"*. These are just warnings — they do not cause any errors. The commands still work.

### Step 9 — Initialize MariaDB (first time only)

```
mariadb-install-db
```

> You will see several lines of text. Wait for `$` to appear.  
> **If you see "mysql.user table already exists"** — MariaDB was already initialized. Skip to Step 10.

---

### Step 10 — Start the MariaDB database server

```
mariadbd-safe --datadir="$PREFIX/var/lib/mysql" &
```

> The `&` runs it in the background. You will see some startup messages, then the `$` prompt returns. **Wait 5–10 seconds** before continuing.

Press **Enter** once more to get a clean `$` prompt.

> **If `mariadbd-safe` is not found**, try the older name:
> ```
> mysqld_safe --datadir="$PREFIX/var/lib/mysql" &
> ```

---

### Step 11 — Open the database console

```
mariadb -u root
```

> You will see `MariaDB [(none)]>` — you are now inside the database.
>
> **If the command is not found**, try:
> ```
> mysql -u root
> ```

---

### Step 12 — Create the NanoClip database and user

You are now at the `MariaDB [(none)]>` prompt. Run **each line below one at a time**, pressing **Enter** after each one and waiting for `Query OK` before continuing:

```sql
CREATE DATABASE IF NOT EXISTS nanoclip CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

```sql
CREATE USER IF NOT EXISTS 'nanoclip'@'localhost' IDENTIFIED BY 'nanoclip123';
```

```sql
CREATE USER IF NOT EXISTS 'nanoclip'@'127.0.0.1' IDENTIFIED BY 'nanoclip123';
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

You are now back at the `$` prompt.

---

### Step 13 — Test that the database works

Run this command to verify the connection:

```
mariadb -h 127.0.0.1 -P 3306 -u nanoclip -pnanoclip123 nanoclip
```

> **Important:** Use `-h 127.0.0.1` (not `localhost`). On modern MariaDB, connecting to `localhost` uses a UNIX socket which may not accept password authentication. Using `127.0.0.1` forces a TCP connection.

If you see `MariaDB [nanoclip]>` — it worked! Type `EXIT;` and press **Enter** to return to `$`.

**If you see "Access denied"**, the user or grants were not saved correctly. Re-run Step 11 and Step 12 to redo the setup, then try again.

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

> Replace `my-secret-key-change-this-please` with any random phrase you like (e.g. `blue-sunset-mountain-2024`).
>
> If you chose a different password in Step 12, replace `nanoclip123` in the DSN above with your password.

---

## Part 6 — Build NanoClip

> **Important for old phones:** Building on a phone takes 10–30 minutes and uses a lot of memory. Plug your phone in before starting. Close all other apps.

### Step 15 — Download the web interface files

```
pnpm install
```

> Takes 5–10 minutes. Wait for `$` before continuing.

---

### Step 16 — Build the web interface (Termux low-memory build)

> **Use this specific command on Termux.** It skips the TypeScript type-check step to avoid running out of memory.

```
NODE_OPTIONS="--max-old-space-size=384" pnpm --filter @nanoclip/ui build:termux
```

> You will see lines starting with `vite v...`. When it says `✓ built in X.Xs`, it is done.
>
> **If you still get an "out of memory" error**, try an even lower limit:
> ```
> NODE_OPTIONS="--max-old-space-size=256" pnpm --filter @nanoclip/ui build:termux
> ```

---

### Step 17 — Build the NanoClip server

```
cd go-server && go build -o nanoclip-build . && mv nanoclip-build nanoclip && cd ..
```

> Nothing will appear for several minutes — that is normal. When `$` appears again, the binary is built.
>
> **If it fails with a memory error**, close all other apps and try again. You can also reduce parallel compilation:
> ```
> cd go-server && GOFLAGS="-p=1" go build -o nanoclip-build . && mv nanoclip-build nanoclip && cd ..
> ```

---

## Part 7 — Run NanoClip

### Step 18 — Start the database (every time you open Termux)

Every time you open a fresh Termux session, start MariaDB first:

```
mariadbd-safe --datadir="$PREFIX/var/lib/mysql" &
```

Wait 5–10 seconds, then press **Enter** for a clean prompt.

---

### Step 19 — Start NanoClip

```
./go-server/nanoclip
```

You will see:
```
[db] connecting to MariaDB...
[db] migrations applied
[server] NanoClip listening on 0.0.0.0:8080
```

When you see **"NanoClip listening"**, it is ready.

---

### Step 20 — Open NanoClip in your browser

1. Open any browser on your phone
2. Type in the address bar: `http://localhost:8080`
3. Press **Go**

You will see the NanoClip login screen. Sign up with any email and password — they are stored only on your phone.

---

## Part 8 — Keep NanoClip running in the background

### Option A — Keep Termux open (simplest)

Swipe down from the top and keep the Termux notification visible. Go to **Battery Settings → Termux → Don't optimize** to prevent Android from killing it.

---

### Option B — Run in background with nohup

```
nohup bash -c 'mariadbd-safe --datadir="$PREFIX/var/lib/mysql"; sleep 10; ./go-server/nanoclip' > ~/nanoclip.log 2>&1 &
```

To check logs: `cat ~/nanoclip.log`  
To stop: `pkill mariadbd; pkill nanoclip`

---

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

| What you want to do | Command |
|---|---|
| Start the database | `mariadbd-safe --datadir="$PREFIX/var/lib/mysql" &` |
| Start NanoClip | `./go-server/nanoclip` |
| Stop everything | `pkill mariadbd; pkill nanoclip` |
| View live log | `tail -f ~/nanoclip.log` |
| Go into NanoClip folder | `cd ~/nanoclip-v2` |
| Open in browser | `http://localhost:8080` |

---

## Troubleshooting

### "mysql.user table already exists" during mariadb-install-db

MariaDB was already initialized. **Skip Step 9** and go directly to Step 10.

---

### "Deprecated program name" warnings

Messages like *"Deprecated program name. It will be removed in a future release, use 'mariadb' instead"* are **safe to ignore**. The commands still work. The guide above already uses the new names where possible.

---

### "Access denied for user 'nanoclip'@'localhost'" in Step 13

This happens because `localhost` uses UNIX socket authentication on newer MariaDB. Use `127.0.0.1` to force TCP instead:

```
mariadb -h 127.0.0.1 -P 3306 -u nanoclip -pnanoclip123 nanoclip
```

If you still get Access Denied, redo Step 12 to recreate the user and grants.

---

### "Out of memory" or "SIGABRT" during Step 16

Your phone ran out of memory building the web interface. Try:

1. Close all other apps
2. Use an even lower memory limit:
   ```
   NODE_OPTIONS="--max-old-space-size=256" pnpm --filter @nanoclip/ui build:termux
   ```
3. If it still fails, wait a minute and try again — Android sometimes frees memory after a moment

---

### "Can't connect to local MySQL server" after starting NanoClip

MariaDB is not running. Start it:

```
mariadbd-safe --datadir="$PREFIX/var/lib/mysql" &
```

Then wait 10 seconds and restart NanoClip.

---

### "command not found: go" or "command not found: mariadb"

The packages did not install correctly. Reinstall:

```
pkg install -y golang nodejs-lts mariadb git
```

---

### Page does not load in browser

- Use `http://` not `https://`
- Try `http://127.0.0.1:8080` instead of `localhost`
- Make sure NanoClip is running — you should see "NanoClip listening" in Termux

---

### "signal: killed" or "out of memory" during Step 17 (Go build)

```
cd go-server && GOFLAGS="-p=1" go build -o nanoclip-build . && mv nanoclip-build nanoclip && cd ..
```

`-p=1` limits the build to one CPU core at a time, using much less memory.

---

## Configuration options

Edit `~/.nanoclip-v2/.env` with:

```
nano .env
```

| Setting | What it does | Default |
|---|---|---|
| `MARIADB_DSN` | Database connection string | (required for MariaDB) |
| `GO_PORT` | Port for the web interface | `8080` |
| `JWT_SECRET` | Security key — set to any random phrase | auto-generated |
| `NANOCLIP_DATA_DIR` | Where SQLite is stored (if not using MariaDB) | `~/.nanoclip/` |

After editing `.env`, restart NanoClip (Ctrl+C then run Step 19 again).

---

## Updating NanoClip

When a new version is released:

```
cd ~/nanoclip-v2
git pull
pnpm install
NODE_OPTIONS="--max-old-space-size=384" pnpm --filter @nanoclip/ui build:termux
cd go-server && go build -o nanoclip-build . && mv nanoclip-build nanoclip && cd ..
```

Then restart the server.

---

## Need help?

Open an issue at: **https://github.com/artistrydat/nanoclip-v2/issues**

Please include:
- Your Android version (Settings → About phone → Android version)
- The exact error message you see
- Which step number you are on

---

## License

MIT — free to use, modify, and distribute.
