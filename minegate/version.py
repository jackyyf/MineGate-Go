#!/usr/bin/env python
# -*- encoding: utf-8 -*-

code_template = """\
package minegate

var version_short string = "{version_short}"
var version_full string = "{version_full}"
"""

import sys

if len(sys.argv) != 3:
	print >>sys.stderr, 'Usage: %s version_short version_full' % sys.argv[0]
	sys.exit(1)

with open('minegate/version.go', 'wb') as f:
	f.write(code_template.format(
		version_short=sys.argv[1].replace('"', '\\"'),
		version_full=sys.argv[2].replace('"', '\\"'),
		))