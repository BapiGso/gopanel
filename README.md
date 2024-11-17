
![preview](./assets/img/preview2.png)


![license](https://img.shields.io/github/license/BapiGso/gopanel)
![last - commit](https://img.shields.io/github/last-commit/BapiGso/gopanel)
![stars - gopanel](https://img.shields.io/github/stars/bapigso/gopanel?style=social)
![forks - gopanel](https://img.shields.io/github/forks/bapigso/gopanel?style=social)
![CodeQL](https://github.com/BapiGso/gopanel/workflows/CodeQL/badge.svg)

---

## ABOUT

**gopanel** It is a management panel written in Go language with zero dependencies, super simple deployment and very simple functions.

❗Entertainment project, do not use in production environment

#### INSTALL

```bash
bash <(curl -s https://raw.githubusercontent.com/BapiGso/gopanel/master/shell/install_gopanel.sh)
```

Or, download the appropriate version for your platform from the [Releases](https://github.com/BapiGso/gopanel/releases) page and run it.

#### UNINSTALL

```bash
bash <(curl -s https://raw.githubusercontent.com/BapiGso/gopanel/master/shell/uninstall_gopanel.sh)
```

#### FUNCTION
- Panel security entrance
- Server monitoring([github.com/shirou/gopsutil](https://github.com/shirou/gopsutil))
- cron ([github.com/go-co-op/gocron/v2](https://github.com/go-co-op/gocron))
- webssh ([golang.org/x/crypto/ssh](https://golang.org/x/crypto/ssh))
- webdav server ([golang.org/x/net/webdav](https://golang.org/x/net/webdav))
- web file editor
- caddy manage ([github.com/caddyserver/caddy/v2](https://github.com/caddyserver/caddy))
- frps manage ([github.com/fatedier/frp](https://github.com/fatedier/frp))
- frpc manage ([github.com/fatedier/frp](https://github.com/fatedier/frp))
- ~~UnblockNeteaseMusic~~ ([github.com/cnsilvan/UnblockNeteaseMusic](https://github.com/cnsilvan/UnblockNeteaseMusic))
- docker manage ([github.com/docker/docker](https://github.com/docker/docker))
- firewall ([github.com/google/nftables](https://github.com/google/nftables))


#### TODO

- headscale ([github.com/juanfont/headscale](https://github.com/juanfont/headscale))

## LICENSE

released under the [Apache License](https://github.com/BapiGso/gopanel/blob/master/LICENSE).

## Acknowledgments

I would like to extend my heartfelt thanks to the gopher developers and the open-source community for their invaluable contributions. Your efforts have made this project possible.
