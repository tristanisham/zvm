// This is the (dummy) fallback implementation to use if the `noAutoUpgrades` tag was passed for building.
//go:build noAutoUpgrades

package meta

const (
	NoAutoUpgrades BuildFlag = true
)
