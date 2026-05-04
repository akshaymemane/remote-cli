import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'remote-cli',
  description: 'Self-hosted mobile control plane for Claude Code sessions across your machines.',
  base: '/remote-cli/',
  cleanUrls: true,
  lastUpdated: true,
  metaChunk: true,
  head: [
    ['meta', { name: 'theme-color', content: '#111827' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:title', content: 'remote-cli' }],
    ['meta', { property: 'og:description', content: 'Control Claude Code sessions across your machines from a phone.' }]
  ],
  themeConfig: {
    siteTitle: 'remote-cli',
    search: {
      provider: 'local'
    },
    nav: [
      { text: 'Guide', link: '/quickstart' },
      { text: 'Deploy', link: '/relay-deployment' },
      { text: 'Troubleshooting', link: '/troubleshooting' },
      { text: 'GitHub', link: 'https://github.com/akshaymemane/remote-cli' }
    ],
    sidebar: [
      {
        text: 'Getting Started',
        items: [
          { text: 'Overview', link: '/' },
          { text: 'Quickstart', link: '/quickstart' },
          { text: 'Choosing Your Relay URL', link: '/choosing-relay-url' }
        ]
      },
      {
        text: 'Install And Deploy',
        items: [
          { text: 'Relay Deployment', link: '/relay-deployment' },
          { text: 'Agent Install', link: '/agent-install' },
          { text: 'Service And Autostart', link: '/service-autostart' },
          { text: 'Troubleshooting', link: '/troubleshooting' }
        ]
      },
      {
        text: 'Internals',
        items: [
          { text: 'Architecture', link: '/architecture' },
          { text: 'Protocol', link: '/protocol' },
          { text: 'Development', link: '/development' },
          { text: 'Roadmap', link: '/roadmap' }
        ]
      }
    ],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/akshaymemane/remote-cli' }
    ],
    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2026 remote-cli contributors'
    },
    editLink: {
      pattern: 'https://github.com/akshaymemane/remote-cli/edit/master/docs/:path',
      text: 'Edit this page on GitHub'
    }
  }
})
