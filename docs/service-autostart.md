# Service And Autostart

Alpha status: remote-cli does not yet include service management commands.

For now, the supported path is:

```bash
remote-cli run
```

Keep that process running for the device to stay online.

## Why Autostart Matters

Pairing saves credentials, but the device only appears online while the agent process is connected to the relay.

If the terminal closes, machine sleeps, or process exits, the relay marks the device offline.

## Planned Commands

Future versions may include:

```bash
remote-cli service install
remote-cli service start
remote-cli service stop
remote-cli service status
remote-cli logs
```

## Linux: systemd User Service Example

This is a manual example for advanced users.

Create:

```text
~/.config/systemd/user/remote-cli-agent.service
```

Example unit:

```ini
[Unit]
Description=remote-cli agent
After=network-online.target

[Service]
ExecStart=%h/.local/bin/remote-cli run
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
```

Enable:

```bash
systemctl --user daemon-reload
systemctl --user enable --now remote-cli-agent
```

Logs:

```bash
journalctl --user -u remote-cli-agent -f
```

## macOS: launchd Notes

launchd support is not documented as a supported install path yet.

For now, run:

```bash
remote-cli run
```

in a terminal, or use your own launchd plist if you are comfortable debugging it.

## Public Alpha Position

Public alpha docs should be honest:

- pairing does not install a service
- `remote-cli run` must stay running
- service helpers are planned but not ready
