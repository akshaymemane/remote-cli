# Service And Autostart

Pairing saves credentials, but the device only appears online while the agent process is connected to the relay.

For quick manual testing, run:

```bash
remote-cli run
```

Keep that process running for the device to stay online. For day-to-day use, install the background service.

## Supported Commands

```bash
remote-cli service install
remote-cli service uninstall
remote-cli service start
remote-cli service stop
remote-cli service logs
```

Service management currently supports:

- macOS via launchd
- Linux via systemd user services

There is no `remote-cli service status` command yet. Use service logs and the PWA device status for now.

## Install

```bash
remote-cli service install
```

This writes a service file for the current user and starts the agent.

The command uses the current `remote-cli` executable path, so install the binary in its final location before installing the service.

## Logs

```bash
remote-cli service logs
```

On Linux, this follows the systemd user journal. On macOS, this follows:

```text
~/Library/Logs/remote-cli.log
```

## Start And Stop

```bash
remote-cli service stop
remote-cli service start
```

Stopping the service makes the device go offline in the PWA. Starting it reconnects the agent.

## Uninstall

```bash
remote-cli service uninstall
```

This stops and removes the service file. It does not unpair the device.

## Linux Details

The generated service path is:

```text
~/.config/systemd/user/remote-cli.service
```

The service runs:

```text
remote-cli run
```

Equivalent manual commands:

```bash
systemctl --user start remote-cli
systemctl --user stop remote-cli
journalctl --user -u remote-cli -f --no-pager
```

## macOS Details

The generated launchd plist path is:

```text
~/Library/LaunchAgents/com.remote-cli.agent.plist
```

The service writes stdout and stderr to:

```text
~/Library/Logs/remote-cli.log
```

Equivalent manual commands:

```bash
launchctl start com.remote-cli.agent
launchctl stop com.remote-cli.agent
tail -f ~/Library/Logs/remote-cli.log
```

## Common Issues

If the service starts but the device stays offline:

- run `remote-cli status` and confirm the agent is paired
- confirm the stored relay URL is reachable from that machine
- confirm Claude Code works with `claude --print "Reply with OK"`
- check `remote-cli service logs`

If the binary was moved after service installation, reinstall the service:

```bash
remote-cli service uninstall
remote-cli service install
```
