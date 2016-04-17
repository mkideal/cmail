# smtpd [![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/mkideal/pkg/master/LICENSE)

## License

[The MIT License (MIT)](https://raw.githubusercontent.com/mkideal/pkg/master/LICENSE)

## Install

```shell
go get github.com/pkg/smtpd
```

## Usage

```shell
smtpd [OPTIONS]
```

Or

```shell
sudo smtpd [OPTIONS]
```

**Options**

```shell
  -h, --help
      display help

  --config[=$SMTPD_CONFIG_FILE]
      config file name

  -H, --host[=0.0.0.0]
      listening host

  -p, --port[=25]
      listening port

  --debug[=false]
      enable debug mode

  --db-source[=$SMTPD_DB_SOURCE]
      mysql db source

  --dn, --domain-name
      my domain name

  --max-session-size[=32768]
      max size of sessions(less than max size of open files)

  --max-error-size[=3]
      max size of errors

  --max-buffer-szie[=6553600]
      max size of buffer

  --max-recipients[=256]
      max size of recipients
```
