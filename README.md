# powerdns-consul

powerdns-consul is a utility written in [Go](https://golang.org) that allows you
to use the [Consul](https://consul.io) key value store as a backend for
[PowerDNS](https://www.powerdns.com) through the [Pipe backend](https://doc.powerdns.com/md/authoritative/backend-pipe/).

Please note that this project is still work in progress. Further usage
instructions will be added when the code is actually usable, is covered by
some tests and has proven to work in the Wild(tm).

Some of the PowerDNS-related code is inspired by [mindreframer's work](https://github.com/mindreframer/golang-stuff/blob/master/github.com/youtube/vitess/go/cmd/zkns2pdns/pdns.go).

## Building

- Clone the repository in your `$GOPATH/src/github.com/Shark/powerdns-consul`
- Run `go get .`
- Run `go build .`

## Usage

WIP.

## Contributing
1. Fork it!
2. Create your feature branch: `git checkout -b my-new-feature`
3. Commit your changes: `git commit -am 'Add some feature'`
4. Push to the branch: `git push origin my-new-feature`
5. Submit a pull request! :)

## History

- v0.0.1 (2016-04-20): initial version

## License

This project is licensed under the MIT License. See LICENSE for details.
