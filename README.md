<div align="center">

<img src="assets/logo-512.png" width="96" alt="tinyurl">

# tinyurl

**A URL shortener in Go that fits in one file.**

[![CI](https://github.com/marcus-beauregard/tinyurl/actions/workflows/ci.yml/badge.svg)](https://github.com/marcus-beauregard/tinyurl/actions/workflows/ci.yml)
[![License: GPL-2.0-or-later](https://img.shields.io/badge/license-GPL--2.0--or--later-blue.svg)](LICENSE)

</div>

## Quick start

```bash
go run .
```

The server listens on **:8080**. Mint a short link:

```bash
curl -s -X POST localhost:8080 -d '{"url":"https://example.com"}'
# {"short":"http://localhost:8080/1a2b3c4d"}
```

Follow it — a 302, with the original URL in ```Location```:

```bash
curl -i localhost:8080/1a2b3c4d
```

## What it does

- POST a JSON body with a ```url``` key, get ```{"short": "..."}``` back.
- GET ```/<code>``` redirects, or answers 404.
- Codes are 4 random bytes as hex — eight characters.
- State is a map behind a ```sync.RWMutex```, so concurrent POSTs cannot corrupt it.

## Develop

```bash
go test ./...
gofmt -l .        # should print nothing
go vet ./...
```

## Limits, stated plainly

Links live in memory (#7), there is no expiry (#8), no rate limit (#9), and the scheme is hardcoded to ```http``` (#6). Those are open issues, not surprises — pick one up.

## Contributing

Humans and automated agents are both welcome. There is no CLA — every commit carries a [DCO](https://developercertificate.org/) sign-off (**git commit -s**).

Start with a [good first issue](https://github.com/marcus-beauregard/tinyurl/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22), or read [CONTRIBUTING.md](CONTRIBUTING.md).

## Licence

GPL-2.0-or-later — see [LICENSE](LICENSE). Copyright (C) 2026 Marcus Beauregard.
