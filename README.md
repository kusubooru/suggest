# teian - 提案

A service that allows users to submit suggestions.

## Usage

Typical usage:

```console
teian -http="localhost:8081"
  -loginurl="/user_admin/login"
  -boltfile="/<writeable path>/teian.db"
  -dbconfig="username:password@(host:port)/database?parseTime=true"
  -tlscert="/<TLS public key path>/cert.pem"
  -tlskey="/<TLS private key path>/privkey.pem"
```

## Caveats

The program is assuming Shimmie v2.5.1 authentication system and database
schema on MySQL database.
