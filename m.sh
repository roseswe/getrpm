#!/bin/bash
# $Id: m.sh,v 1.1 2026/04/06 06:27:32 ralph Exp $

go fmt
# go fix
# go vet

go build listprodids.go
##go build getrpm.go
go build -ldflags "-X main.compileDate=$(date +%d.%m.%Y)" -o getrpm getrpm.go

strip getrpm listprodids
upx --lzma   getrpm listprodids

cp getrpm ~/bin/

