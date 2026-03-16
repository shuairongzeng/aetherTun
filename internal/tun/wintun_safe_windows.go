//go:build windows

package tun

import (
	"fmt"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wintun"
)

var createAdapterFn = wintun.CreateAdapter

func createAdapterSafe(adapterName string) (adapter *wintun.Adapter, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("wintun adapter creation panicked: %v", recovered)
			adapter = nil
		}
	}()

	return createAdapterFn(adapterName, "Wintun", (*windows.GUID)(nil))
}
