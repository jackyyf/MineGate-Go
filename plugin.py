#!/usr/bin/env python
# -*- encoding: utf-8 -*-

import argparse
import sys
import os
import os.path

code_template = \
    """
package minegate

import (
	{import_packages}
)
"""

os.chdir(os.path.dirname(os.path.abspath(__file__)))

if __name__ == '__main__':
    parser = argparse.ArgumentParser(
        description='Plugin manager for minegate.')
    parser.add_argument('-l', '--list', dest='action', action='store_const', const='list',
                        default='list', help='Show currently added plugin.')
    parser.add_argument('-a', '--add', dest='action', action='store_const', const='add',
                                            help='Add a plugin.')
    parser.add_argument('-d', '--delete', dest='action', action='store_const', const='delete',
                        help='Delete a plugin.')
    parser.add_argument('-g', '--gen', dest='action', action='store_const', const='generate',
                                            help='Generate the dummy golang package.')
    # parser.add_argument('-o', '--output', dest='output', action='store',
    #					help='Specify output file. Works with --gen only.')
    parser.add_argument(
        'packages', nargs='*', help='List of packages.', default=[])
    args = parser.parse_args()
    if args.action in ['add', 'delete']:
        if not args.packages:
            print 'No package specified.'
            sys.exit(1)
        packages = []
        try:
            with open('.plugins', 'r') as f:
                packages = map(lambda s: s.strip(), f.readlines())
        except IOError:
            pass
        if args.action == 'add':
            package_set = set(packages)
            package_set.update(args.packages)
            with open('.plugins', 'w') as f:
                f.write('\n'.join(package_set))
        elif args.action == 'delete':
            package_set = set(packages)
            package_set = package_set.difference(
                args.packages if isinstance(args.packages, (list, tuple)) else [args.packages])
            with open('.plugins', 'w') as f:
                f.write('\n'.join(package_set))
        sys.exit(0)
    elif args.action == 'generate':
        if args.packages:
            print 'Warning: packages are not used in this command.'
        try:
            with open('.plugins', 'r') as f:
                packages = map(lambda s: s.strip(), f.readlines())
        except IOError:
            print 'No plugins added.'
            sys.exit(1)
        if packages:
            with open('minegate/plugin.go', 'w') as f:
                code_data = code_template.format(
                    import_packages='\n'.join(
                        map(lambda s: '\t_ "%s"' %
                            s.replace('"', '\\"'), packages)
                    )
                )
                f.write(code_data)
        else:
            print 'No plugins added.'
    elif args.action == 'list':
        if args.packages:
            print 'Warning: packages are not used in this command.'
        try:
            with open('.plugins', 'r') as f:
                packages = sorted(map(lambda s: s.strip(), f.readlines()))
        except IOError:
            print '0 plugins installed.'
            sys.exit(0)
        print '%d plugin%s installed: ' % (len(packages), 's'[len(packages) == 1:])
        print '\n'.join(packages)
