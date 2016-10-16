# teian - 提案

A service that allows users to submit suggestions.

## Installation

```console
go get -u github.com/kusubooru/teian
```

## Usage

Local example:

```console
teian -dbconfig=username:password@(localhost:3306)/database?parseTime=true
```

Live example:

```console
teian -http="localhost:8081"
  -loginurl="/user_admin/login"
  -boltfile="/<writeable path>/teian.db"
  -dbconfig="username:password@(host:port)/database?parseTime=true"
  -tlscert="/<TLS public key path>/cert.pem"
  -tlskey="/<TLS private key path>/privkey.pem"
```

## Notes

The program needs data from MySQL for users, authentication and some common
configuration. This is handled by
[kusubooru/shimmie](https://github.com/kusubooru/shimmie) which is vendored. The database
schema should be the same as the one used in v2.5.1 of the
[Shimmie2](https://github.com/shish/shimmie2) project.

The suggestions are stored in a file teian.db using
[boltdb](https://github.com/boltdb/bolt).
