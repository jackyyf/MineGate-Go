#!/usr/bin/env python
# -*- encoding: utf-8 -*-


import os
import sys
import qiniu

ak = os.environ['QINIU_AK']
sk = os.environ['QINIU_SK']
bucket = os.environ['QINIU_BUCKET']

def sizefmt(num, suffix='B'):
	for unit in ['','Ki','Mi','Gi','Ti','Pi','Ei','Zi']:
		if abs(num) < 980.0:
			num = ('%.3f' % num)[:5].rstrip('.')
			return "%s %s%s" % (num, unit, suffix)
		num /= 1024.0
	return "%.1f%s%s" % (num, 'Yi', suffix)

def handler(now, tot):
	print 'Uploaded: %s / %s [%5.2f%%]' % (sizefmt(now), sizefmt(tot), now * 100. / tot )

notify = ''

if len(sys.argv) != 3:
	print >>sys.stderr, '%s file key' % sys.argv[0]
	exit(1)

q = qiniu.Auth(ak, sk)

fname, key = sys.argv[1], sys.argv[2]
policy = dict()

if notify:
	policy['callbackUrl'] = notify

policy['insertOnly'] = False

token = q.upload_token(bucket, key, expires=1200, policy=policy)

qiniu.put_file(up_token=token, key=key, file_path=fname, progress_handler=handler)