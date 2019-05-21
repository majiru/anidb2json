# AniDB2Json

Creates json database file from a directory of media.

## Command Line Utility

`./anidb2json titledb mediadir [cachedir]`

so for example

`./anidb2json titles.xml /home/user/media /tmp/anidbcache`

## Library

All of the structs and functions used by the command line utility are in the main package.
`cmd/anidb2json/main.go` provides an example.
