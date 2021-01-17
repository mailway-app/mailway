#!/usr/bin/env python3

import sys
import os
import io
import tempfile
import subprocess
import logging

emailfile = sys.argv[1]
ftmp = tempfile.NamedTemporaryFile(delete=False)

try:
    email = io.open(emailfile, mode='r', buffering=-1)
    res = subprocess.call(["spamc"], stdout=ftmp, stderr=sys.stderr,
                          stdin=email)
    ftmp.close()
    if res != 0:
        logging.error("tempfile %s" % ftmp.name)
        raise Exception("spamc terminated with non zero")

    os.replace(ftmp.name, emailfile)
    exit(0)
except Exception as e:
    raise e
    exit(1)
