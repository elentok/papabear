# Papa Bear

![Papa Bear logo](docs/logo-512.png)

Papa Bear is a daemon that lets parents remotely control screen time on Linux machines via Telegram
or SSH. It tracks active session time via `loginctl`, enforces limits by locking the screen and
account, and sends desktop notifications and TTS alerts when time is running low.

## Requirements

- Ubuntu (systemd + systemd-logind)
- A Telegram bot token (create one via [@BotFather](https://t.me/BotFather))
- Runtime apt dependencies are installed by `setup`: `sudo`, `libnotify-bin`, `pulseaudio-utils`,
  `python3-venv`, `gnome-shell-extension-appindicator`, `python3-gi`, `gir1.2-gtk-3.0`,
  `gir1.2-ayatanaappindicator3-0.1`

## Install

1. Build or download the binary:

   ```sh
   go build -o papabear
   sudo cp papabear /usr/local/bin/
   ```

2. Run setup (creates system user, config, sudoers, PAM rule, and systemd service):

   ```sh
   sudo papabear setup
   ```

3. Edit the config:

   ```sh
   sudo nano /etc/papabear/config.yaml
   ```

4. Enable and start:
   ```sh
   sudo systemctl enable --now papabear
   ```

## Configuration

`/etc/papabear/config.yaml`:

```yaml
machine_name: "Bob-PC"

telegram:
  bot_token: "TOKEN"
  allowed_chat_ids:
    - 111111111 # get this from scripts/get-chat-id.sh

server:
  listen_addr: "127.0.0.1:3847"

notifications:
  thresholds: [30, 15, 5, 1] # minutes remaining

tts:
  model: "en_US-lessac-medium"

users:
  - name: "bob"
    daily_limit_minutes: 300 # 5 hours
    allowed_hours:
      start: 8 # 8am
      end: 18 # 6pm
```

## Telegram Commands

| Command                      | Effect                                                               |
| ---------------------------- | -------------------------------------------------------------------- |
| `/give [bob] 30m`            | Add 30 minutes to Bob's time                                         |
| `/give [bob] 1h30m`          | Add 1.5 hours                                                        |
| `/lock [bob]`                | Lock Bob's screen and account immediately                            |
| `/unlock [bob] 15m`          | Set Bob's remaining time to 15 minutes and allow login               |
| `/status [bob]`              | Show remaining time, used time, allowed hours, and activity timeline |
| `/hours [bob]`               | Show Bob's allowed hours                                             |
| `/hours [bob] 8-20`          | Set allowed hours to 8am-8pm                                         |
| `/say [bob] Time for dinner` | Speak a message to Bob via TTS                                       |

Duration formats: `15`, `15m`, `1h`, `1h30m`.

The user argument can be omitted when there is one configured user, or when exactly one configured user is active.

`/lock [bob] 15m` still works as a compatibility alias for `/unlock [bob] 15m`, but `/unlock` is preferred.

Using `/give` outside allowed hours automatically creates a temporary override so the child can log in.

## User Commands

These commands are for the child to run on their own machine:

```sh
papabear status   # show remaining screen time, allowed hours, and today's activity
papabear status --compact  # show only remaining screen time
papabear ask      # request more time (notifies parents via Telegram)
papabear ask 30   # request 30 minutes specifically
```

## Admin Commands

```sh
papabear run          # start the daemon (normally via systemd)
papabear setup        # install system dependencies (run as root)
papabear doctor       # check configuration and dependencies
papabear logs         # tail the service logs
papabear give bob 30m # add 30 minutes for bob
papabear lock bob     # lock bob's screen and account immediately
papabear unlock bob 15m  # set bob's remaining time to 15 minutes and allow login
papabear status bob   # show bob's remaining time and activity timeline
papabear hours bob    # show allowed hours for bob
papabear hours bob 8-20  # set allowed hours
papabear hours bob saturday 10-14  # set a specific day's allowed hours
papabear say bob "Time for dinner"  # send a desktop notification and TTS message
papabear completion bash > /etc/bash_completion.d/papabear  # generate bash completion
papabear completion fish > ~/.config/fish/completions/papabear.fish  # generate fish completion
papabear completion zsh > ~/.zsh/completions/_papabear  # generate shell completion
```

For SSH/admin use, the user argument can be omitted for `give`, `lock`, `unlock`, `status`, `hours`, and `say` when there is one configured user, or when exactly one configured user is active.

## HTTP API

```sh
# Request more time (used by `papabear ask`)
curl -X POST "http://127.0.0.1:3847/request-more-time?user=bob&minutes=15"

# Check status (used by `papabear status`)
curl "http://127.0.0.1:3847/status?user=bob"
```

## How It Works

- The daemon polls `loginctl` every 10 seconds to check session state
- Time only counts when the session is active (not locked or idle)
- Daily usage is stored in `/var/lib/papabear/usage.json` and resets at midnight
- Activity transitions (active/locked/idle/offline) are logged to `/var/lib/papabear/log/{user}/YYYY-MM-DD.log`
- When time runs out: screen locks, account access is disabled via `chage -E 0`, and parents are notified
- When time is granted via `/give`, the child receives a desktop notification and TTS announcement
- TTS uses `piper-tts` with the `en_US-lessac-medium` voice model; generated WAV files are cached in `/var/lib/papabear/tts-cache/` so repeated messages play instantly
- `setup` installs `/usr/local/bin/papabear-tray` and autostarts it for configured users to show remaining time from `status --compact`
- Login is enforced via PAM (`pam_exec`) -- the child cannot log in outside allowed hours or with no time remaining
- Parents can grant time or adjust hours at any time via Telegram

## Deploy

```sh
./scripts/deploy.sh myserver.example.com
```

## Logs

```sh
journalctl -u papabear -f
```
