package migrate

import (
	"github.com/cjimti/migration-kit/cfg"
	"github.com/cjimti/migration-kit/driver"
	"github.com/cjimti/migration-kit/tunnel"
)

// Project defines an overall project consisting of
// Databases and Migrations
type Project struct {
	Component     cfg.Component
	Databases     map[string]cfg.Database  // map of database machine names to databases
	Migrations    map[string]cfg.Migration // map of migration machine names to migrations
	Tunnels       map[string]cfg.Tunnel    // map of tunnels
	driverManager driver.Manager
	tunnelManager tunnel.Manager
}
