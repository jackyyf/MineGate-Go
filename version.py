#!/usr/bin/env python
# -*- encoding: utf-8 -*-

code_template = """\
package minegate

import (
	"fmt"
	"runtime"
)

var version_short string = fmt.Sprintf("{version_short}", runtime.GOARCH, runtime.GOOS)
var version_full string = fmt.Sprintf("{version_full}", runtime.GOARCH, runtime.GOOS)
"""

import sys

if len(sys.argv) != 3:
	print >>sys.stderr, 'Usage: %s version_short version_full' % sys.argv[0]
	sys.exit(1)

with open('minegate/version.go', 'wb') as f:
	f.write(code_template.format(
		version_short=sys.argv[1].replace('"', '\\"').replace('%', '%%').replace('{{.Arch}}', '%[1]s').replace('{{.OS}}', '%[2]s'),
		version_full=sys.argv[2].replace('"', '\\"').replace('%', '%%').replace('{{.Arch}}', '%[1]s').replace('{{.OS}}', '%[2]s'),
		))