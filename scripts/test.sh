#!/bin/sh

go test -v -race $(go list ./...)
