#!/usr/bin/env python
# -*- encoding: utf-8 -*-

code_template = """\
package minegate

import (
	"fmt"
	"runtime"
)

var version_short string = {version_short}
var version_full string = {version_full}
"""

import sys

def gofmt(s):
	s = s.replace('"', '\\"')
	if '{{.Arch}}' in s or '{{.OS}}' in s:
		return 'fmt.Sprintf("{s}", runtime.GOARCH, runtime.GOOS)'.format(s=s.replace('%', '%%').replace('{{.Arch}}', '%[1]s').replace('{{.OS}}', '%[2]s'))
	else:
		return '"' + s + '"'

if len(sys.argv) != 3:
	print >>sys.stderr, 'Usage: %s version_short version_full' % sys.argv[0]
	sys.exit(1)

with open('minegate/version.go', 'wb') as f:
	f.write(code_template.format(
		version_short=gofmt(sys.argv[1]),
		version_full=gofmt(sys.argv[2]),
		))