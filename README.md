# gh-proxy-go

_A golang fork of [hunshcn/gh-proxy](https://github.com/hunshcn/gh-proxy)._

## Build

You can just clone this repo and prepare a golang development environment with golang version >= 1.22, then just run `go build` to get the binary file.

You can also use `docker build . -t [tag_name]` to build an docker image.

`ghcr.io/Allenyou1126/gh-proxy-go:[latest-version]` is also available now.

Currently, `1.1.0` is the latest version.

## Use

Run the binary file directly.

It will listen on `127.0.0.1:80` with http scheme on default.

Also, you can use environment variables or `.env` file to change some configurations.

## Configuration

| Configuration      | Description                                                                                                                 | Acceptable value                                                                                                                                                                              | Default value |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------- |
| DEBUG_MODE         | Control whether to enable debug mode.                                                                                       | `1` or `true` or `True` or `TRUE` for enable, and other values for disable                                                                                                                    | `false`       |
| JSDELIVR_MIRROR    | Control whether to use JsDelivr as the mirror, rather than run a proxy provided by this application.                        | `1` or `true` or `True` or `TRUE` for enable, and other values for disable                                                                                                                    | `false`       |
| SKIP_TLS_VERIFYING | Control whether to skip tls verifying. Enable it to solve "x509: certificate signed by unknown authority" in docker.        | `1` or `true` or `True` or `TRUE` for enable, and other values for disable                                                                                                                    | `false`       |
| SIZE_LIMIT         | Control the maximum file size allowed to proxy through this program. If exceeded, A "Found" response will be provided.      | Integer with Byte as unit, or `xxxG`/`xxxM`/`xxxK` (`xxx` **must** be an integer)                                                                                                             | `999G`        |
| SERVER_HOST        | Control the address server listening on.                                                                                    | Any valid address.                                                                                                                                                                            | `127.0.0.1`   |
| SERVER_PORT        | Control the port server listening on.                                                                                       | A valid port number.                                                                                                                                                                          | `80`          |
| WHITE_LIST         | If set, only repos meets the whitelist rules will be allowed to proxy through this program.                                 | A multi-line string, each line should be a rule. A rule should be like `username/reponame`, and you can use wildcard character `*` in rule. If `reponame` be omitted, it will be `username/*` | `""`          |
| BLACK_LIST         | If set, repos meets the blacklist rules will not be allowed to proxy through this program.                                  | The same as `WHITE_LIST`                                                                                                                                                                      | `""`          |
| PASS_LIST          | If set, repos meets the pass list rules will never go through this program. Instead, it will be redirected to original url. | The same as `WHITE_LIST`                                                                                                                                                                      | `""`          |

## Special thanks

- DeepSeek-R1: Used to provide the initial source code from original python version.
- [@hunshcn](https://github.com/hunshcn): Thanks for providing us with such a powerful program.
