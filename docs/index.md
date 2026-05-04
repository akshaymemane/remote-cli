---
layout: home

hero:
  name: remote-cli
  text: Control Claude Code from your phone
  tagline: A self-hosted relay, PWA, and agent for running Claude Code sessions across your laptop, desktop, and Raspberry Pi.
  actions:
    - theme: brand
      text: Start Quickstart
      link: /quickstart
    - theme: alt
      text: Pick A Relay URL
      link: /choosing-relay-url
    - theme: alt
      text: View On GitHub
      link: https://github.com/akshaymemane/remote-cli

features:
  - title: Self-hosted relay
    details: Run the relay yourself, serve the PWA, and route phone and agent traffic through infrastructure you control.
  - title: Multi-device agents
    details: Pair each machine once, keep the agent online, and choose the target device from the PWA.
  - title: Claude Code sessions
    details: The selected agent starts Claude Code locally and streams assistant output back to your phone.
---

## Current Status

remote-cli is a beta-candidate project for technical users. It is useful for self-hosted testing, dogfooding, and early feedback, but it is not production-ready yet.

Start with the [Quickstart](quickstart.md), then read [Choosing Your Relay URL](choosing-relay-url.md). Most setup issues come from using `localhost` where a LAN IP, Tailscale hostname, or public HTTPS domain is needed.
