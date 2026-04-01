# NanoClip — Android / Termux Setup Guide

> **Who this guide is for:** Anyone who wants to run NanoClip on an Android phone using Termux — even if you have never used a terminal or typed a command before.  
> Every step is explained in plain language. Take it slow, copy each command exactly as shown, and you will have NanoClip running on your phone.

---

## What is NanoClip?

NanoClip is an AI agent platform that runs entirely on your Android phone. You create AI agents, assign them to tasks (called *issues*), and they respond, track progress, and work automatically — all without a cloud subscription. Everything runs locally on your device.

---

## Before you start — check your Android version

This guide works on **Android 7 or newer**. Older Androids (5 or 6) may work but some steps may fail. If your phone is very old (2013 or earlier), the build step will take a long time — plug it in and be patient.

**Minimum requirements:**
- Android 7.0 (Nougat) or newer
- At least **2 GB free storage** (the build tools take a lot of space)
- A working internet connection for the first install

---

## Part 1 — Install Termux

Termux is a free terminal app for Android. **Do NOT install it from the Google Play Store** — that version is old and broken. Use F-Droid instead.

### Step 1 — Install F-Droid

1. Open your phone's browser (Chrome, Firefox, etc.)
2. Go to: **https://f-droid.org**
3. Tap the big **"Download F-Droid"** button
4. When the download finishes, tap the notification or open your Downloads folder and tap the file
5. If your phone asks *"Allow install from unknown sources"*, tap **Settings**, then turn on **"Allow from this source"**, then go back and tap **Install**
6. After it installs, tap **Open**

> **What is F-Droid?** It is a free app store for open-source apps. It is safe and trusted by millions of people worldwide.

---

### Step 2 — Install Termux from F-Droid

1. Open the **F-Droid** app you just installed
2. Tap the search icon (magnifying glass) at the bottom
3. Type: `Termux`
4. Tap the result that says **"Termux"** (it has an icon that looks like a black terminal window)
5. Tap **Install**
6. Wait for it to download and install — this may take a minute
7. Tap **Open** when done

> You now have Termux installed. It will show you a black screen with some text — that is completely normal. That is your terminal.

---

### Step 3 — Give Termux storage permission

NanoClip needs to read and write files. Do this once:

1. In Termux, type the following and press **Enter**:
   ```
   termux-setup-storage
   ```
2. A popup will appear asking for **"Storage"** permission — tap **Allow**

> If nothing happens or no popup appears, go to **Android Settings → Apps → Termux → Permissions → Storage → Allow**.

---

## Part 2 — Install the required tools

You need to install several programs inside Termux before NanoClip can run. These commands download and install everything automatically.

### Step 4 — Update Termux's package list

Copy and paste this command into Termux, then press **Enter**:

```
pkg update -y && pkg upgrade -y
```

> This updates Termux itself. It may ask you questions — just press **Enter** to accept defaults. This step can take 3–10 minutes depending on your internet speed. Wait until you see the `$` symbol again before continuing.

---

### Step 5 — Install Go, Node.js, MariaDB, and Git

Type this command and press **Enter**:

```
pkg install -y golang nodejs-lts mariadb git
```

> **What is each tool?**
> - **golang** — the programming language NanoClip's server is written in
> - **nodejs-lts** — needed to build the web interface
> - **mariadb** — the database that stores all your data (recommended for phones)
> - **git** — downloads NanoClip's source code from the internet
>
> This will download and install everything. It will take **5–15 minutes** on a slow connection. Your phone's screen may turn off — that is fine, the install continues in the background.

---

### Step 6 — Install pnpm (the package manager for the web interface)

```
npm install -g pnpm
```

> This installs `pnpm`, a tool that downloads the web interface's dependencies. Wait for the `$` symbol before continuing.

---

## Part 3 — Download NanoClip

### Step 7 — Download the NanoClip source code

```
git clone https://github.com/artistrydat/nanoclip-v2.git
```

> This downloads NanoClip from GitHub to your phone. You will see lines of text scrolling — that is normal. When it finishes, you will see `$` again.

### Step 8 — Enter the NanoClip folder

```
cd nanoclip-v2
```

> `cd` means "change directory" — it moves you into the NanoClip folder. You must run all future commands from inside this folder.

---

## Part 4 — Set up the database (MariaDB)

This is the most important step. MariaDB is the database that stores everything: your agents, issues, conversations, and settings. On Android, MariaDB works better than SQLite (the default), especially on older devices.

### Step 9 — Initialize MariaDB (first time only)

This sets up the database storage on your phone. Type:

```
mysql_install_db
```

> This creates the database files in Termux's home directory. You will see several lines of text. Wait for `$` to appear.
>
> **If you see an error like "already exists"** — that means MariaDB was already initialized. Skip to Step 10.

---

### Step 10 — Start the MariaDB database server

```
mysqld_safe --datadir="$PREFIX/var/lib/mysql" &
```

> The `&` at the end means "run in the background". You will see some startup messages, then the `$` prompt returns. MariaDB is now running.
>
> **Wait about 5 seconds** before continuing — the database needs a moment to fully start.
>
> **On very old Android phones (Android 7 or earlier):** if you get an error about `--skip-mysqlx`, use this command instead:
> ```
> mysqld --user=$(whoami) --datadir="$PREFIX/var/lib/mysql" --skip-grant-tables &
> ```
> Then press **Enter** once more to get the `$` prompt back.

---

### Step 11 — Open the database console

```
mysql -u root
```

> This opens the MariaDB command prompt. You will see `MariaDB [(none)]>` — this means you are now inside the database. Do not close Termux.
>
> **If you see "Access denied"**, try:
> ```
> mysql -u root --skip-password
> ```

---

### Step 12 — Create the NanoClip database and user

You are now inside the database console (you see `MariaDB [(none)]>`). Copy and paste **each line below one at a time**, pressing **Enter** after each one:

```sql
CREATE DATABASE IF NOT EXISTS nanoclip CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

Press Enter, wait for `Query OK`, then:

```sql
CREATE USER IF NOT EXISTS 'nanoclip'@'localhost' IDENTIFIED BY 'nanoclip123';
```

Press Enter, wait for `Query OK`, then:

```sql
CREATE USER IF NOT EXISTS 'nanoclip'@'127.0.0.1' IDENTIFIED BY 'nanoclip123';
```

Press Enter, wait for `Query OK`, then:

```sql
GRANT ALL PRIVILEGES ON nanoclip.* TO 'nanoclip'@'localhost';
```

Press Enter, wait for `Query OK`, then:

```sql
GRANT ALL PRIVILEGES ON nanoclip.* TO 'nanoclip'@'127.0.0.1';
```

Press Enter, wait for `Query OK`, then:

```sql
FLUSH PRIVILEGES;
```

Press Enter, wait for `Query OK`, then type:

```sql
EXIT;
```

Press Enter to leave the database console. You should be back at the regular `$` prompt.

> **What did we just do?**  
> We created a database called `nanoclip` and a user called `nanoclip` with the password `nanoclip123`. NanoClip will use these to store all its data.
>
> **You can change `nanoclip123` to any password you like** — just remember it for Step 14.

---

### Step 13 — Test that the database works

Make sure you can connect with the new user:

```
mysql -u nanoclip -pnanoclip123 nanoclip
```

> If you see `MariaDB [nanoclip]>`, everything worked. Type `EXIT;` and press **Enter** to go back to the `$` prompt.
>
> **If you see "Access denied"** — double-check that you typed the commands in Step 12 exactly as shown, including the semicolons (`;`) at the end of each line.

---

## Part 5 — Configure NanoClip

### Step 14 — Create the settings file

NanoClip reads its settings from a file called `.env`. Create it now:

```
cat > .env << 'ENVEOF'
MARIADB_DSN=nanoclip:nanoclip123@tcp(127.0.0.1:3306)/nanoclip?charset=utf8mb4&parseTime=True&loc=UTC
JWT_SECRET=my-secret-key-change-this-please
GO_PORT=8080
ENVEOF
```

> **Important:** If you changed the password in Step 12, replace `nanoclip123` in the command above with your chosen password.
>
> **What does each line mean?**
> - `MARIADB_DSN` — tells NanoClip how to connect to MariaDB (the database address, user name, and password)
> - `JWT_SECRET` — a secret key used for security. Change `my-secret-key-change-this-please` to any random words you like (e.g., `purple-elephant-runs-fast-2024`)
> - `GO_PORT` — the port (address number) your phone's browser will use to open NanoClip

---

## Part 6 — Build NanoClip

This step compiles NanoClip from source code into a program your phone can run. **This will take time** — 10 to 30 minutes on a phone. Plug your phone in before starting.

### Step 15 — Download the web interface files

```
pnpm install
```

> This downloads all the pieces needed to build the web interface. You will see a lot of text. Wait for `$` to return. This may take 5–10 minutes.

---

### Step 16 — Build the web interface

```
pnpm --filter @nanoclip/ui build
```

> This compiles the web interface into files the server can serve. You will see lines starting with `vite v...` and then a summary showing file sizes. When it says `✓ built in X.Xs`, it is done.
>
> **If you see an error about memory** on an old phone, try:
> ```
> NODE_OPTIONS="--max-old-space-size=512" pnpm --filter @nanoclip/ui build
> ```

---

### Step 17 — Build the NanoClip server

```
cd go-server && go build -o nanoclip . && cd ..
```

> This compiles the server. You will see nothing for several minutes — that is normal. When the `$` appears again, it is finished. The binary (`nanoclip`) is now inside the `go-server` folder.
>
> **On very old phones with limited RAM:** if the build fails with a memory error, try closing all other apps and running the command again.

---

## Part 7 — Run NanoClip

### Step 18 — Start the database (every time you restart Termux)

Every time you open a fresh Termux session, you need to start MariaDB first:

```
mysqld_safe --datadir="$PREFIX/var/lib/mysql" &
```

Wait 5 seconds, then press **Enter** once more.

---

### Step 19 — Start NanoClip

```
./go-server/nanoclip
```

> You will see startup messages like:
> ```
> [db] connecting to MariaDB...
> [db] migrations applied
> [server] NanoClip listening on 0.0.0.0:8080
> ```
> When you see **"NanoClip listening"**, the server is ready.

---

### Step 20 — Open NanoClip in your browser

1. Open any browser on your phone (Chrome, Firefox, Brave, etc.)
2. In the address bar, type exactly:
   ```
   http://localhost:8080
   ```
3. Press **Go** or **Enter**

You should see the NanoClip login screen. **Sign up** with any email and password — they are stored only on your phone.

---

## Part 8 — Keep NanoClip running in the background

By default, NanoClip stops when you close Termux. Here is how to keep it running.

### Option A — Keep Termux open (simplest)

Swipe down from the top of your screen and pull the Termux notification to keep it running. On most Androids, going to **Battery Settings → Termux → Don't optimize** prevents the OS from killing it.

---

### Option B — Run in background with nohup

Instead of Step 19, use this command to run NanoClip detached from the terminal:

```
nohup bash -c 'mysqld_safe --datadir="$PREFIX/var/lib/mysql"; sleep 5; ./go-server/nanoclip' > ~/nanoclip.log 2>&1 &
```

> NanoClip will now run in the background. To check if it is running:
> ```
> cat ~/nanoclip.log
> ```
>
> To stop it:
> ```
> pkill mysqld; pkill nanoclip
> ```

---

### Option C — Auto-start when your phone boots (advanced)

Install **Termux:Boot** from F-Droid (search for "Termux:Boot"). Then create an auto-start script:

```
mkdir -p ~/.termux/boot
cat > ~/.termux/boot/start-nanoclip.sh << 'BOOTEOF'
#!/data/data/com.termux/files/usr/bin/bash
cd ~/nanoclip-v2
mysqld_safe --datadir="$PREFIX/var/lib/mysql" &
sleep 8
./go-server/nanoclip >> ~/nanoclip.log 2>&1
BOOTEOF
chmod +x ~/.termux/boot/start-nanoclip.sh
```

> After this, NanoClip will start automatically every time you reboot your phone.

---

## Quick reference — commands to know

| What you want to do | Command |
|---|---|
| Start the database | `mysqld_safe --datadir="$PREFIX/var/lib/mysql" &` |
| Start NanoClip | `./go-server/nanoclip` |
| Stop everything | `pkill mysqld; pkill nanoclip` |
| View live log | `tail -f ~/nanoclip.log` |
| Go into NanoClip folder | `cd ~/nanoclip-v2` |
| Open NanoClip in browser | Go to `http://localhost:8080` |

---

## Troubleshooting

### "command not found: go" or "command not found: mysql"

The packages did not install correctly. Run Step 5 again:
```
pkg install -y golang nodejs-lts mariadb git
```

---

### "Can't connect to local MySQL server"

MariaDB is not running. Start it (Step 18):
```
mysqld_safe --datadir="$PREFIX/var/lib/mysql" &
```

---

### "Access denied for user 'nanoclip'"

The password in `.env` does not match what you set in Step 12. Open `.env`:
```
nano .env
```
Check that `nanoclip123` (or your chosen password) matches what you used in Step 12. Press **Ctrl+X** to close nano.

---

### The build fails with "signal: killed" or "out of memory"

Your phone ran out of memory during the build. Try:

1. Close all other apps
2. Rerun the build command
3. If it keeps failing, try building with less memory usage:
   ```
   GOFLAGS="-p=1" go build -o nanoclip .
   ```
   (Run this from inside the `go-server` folder)

---

### Page does not load in browser

- Make sure you typed `http://localhost:8080` (not `https://`)
- Make sure NanoClip is running (you should see the "NanoClip listening" message in Termux)
- Try `http://127.0.0.1:8080` instead

---

### MariaDB takes a long time to start on old Android

This is normal on devices with Android 7 or older and slow storage. Wait **15–20 seconds** after starting MariaDB before starting NanoClip.

---

### "Plugin not found" or pnpm errors during build

Run these commands to clear the cache and retry:
```
pnpm store prune
pnpm install --force
pnpm --filter @nanoclip/ui build
```

---

## Configuration options

All settings go in the `.env` file in the `nanoclip-v2` folder. Edit with:

```
nano .env
```

| Setting | What it does | Default |
|---|---|---|
| `MARIADB_DSN` | Database connection string | (required for MariaDB) |
| `GO_PORT` | Port for the web interface | `8080` |
| `JWT_SECRET` | Security key — set to any random phrase | auto-generated |
| `NANOCLIP_DATA_DIR` | Where SQLite database is stored (if not using MariaDB) | `~/.nanoclip/` |

After editing `.env`, restart NanoClip (stop it with **Ctrl+C** and run Step 19 again).

---

## How to update NanoClip later

When a new version is released, update like this:

```
cd ~/nanoclip-v2
git pull
pnpm install
pnpm --filter @nanoclip/ui build
cd go-server && go build -o nanoclip . && cd ..
```

Then restart the server (Ctrl+C to stop, then run `./go-server/nanoclip` again).

---

## Need help?

Open an issue at: **https://github.com/artistrydat/nanoclip-v2/issues**

Please include:
- Your Android version (Settings → About phone → Android version)
- The exact error message you see
- Which step you are on

---

## License

MIT — free to use, modify, and distribute.
