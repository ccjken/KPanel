//go:build !linux

package main

import (
	"github.com/kejilion/kejilion-panel/internal/contract"
	"os"
)

func publishNodeCheckStatus(contract.ServiceCheckSummary) error { return os.ErrPermission }
