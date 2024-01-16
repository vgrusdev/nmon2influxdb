#!/bin/bash

export GOOS=linux
export GOARCH=ppc64le

/usr/local/go/bin/go build -v

