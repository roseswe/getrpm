#!/bin/bash
# $Id: m.sh,v 1.2 2026/04/22 07:28:50 ralph Exp $

rm getrpm listprodids

go fmt
# go fix
# go vet

go build -ldflags "-X main.compileDate=$(date +%d.%m.%Y)" -o listprodids listprodids.go
##go build getrpm.go
go build -ldflags "-X main.compileDate=$(date +%d.%m.%Y)" -o getrpm getrpm.go

strip getrpm listprodids
upx --lzma   getrpm listprodids

cp getrpm ~/bin/

getrpm -V
./listprodids -V
