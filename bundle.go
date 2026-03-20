package main

import "embed"

// deploymentBundle contains the static deployment assets shipped inside the binary.
//
//go:embed all:compose/** all:helm/gochat/** all:scripts/**
var deploymentBundle embed.FS
