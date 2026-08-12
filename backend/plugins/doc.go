// Package plugins holds regression tests for the static plugin metadata
// bundled under backend/plugins (manifests, config schemas and UI schemas).
//
// The runtime contract for each plugin is defined by the plugin process
// (see backend/plugin-demo/pluginv1). The static schema files in this tree are
// shipped as metadata/documentation, so this package guards them against
// drifting away from that contract.
package plugins
