#!/usr/bin/env python3

import time
import logging
from os import listdir
from os.path import isdir, join
from subprocess import Popen, PIPE
from shutil import unpack_archive

DISTDIR = "modules/"
TIMEOUT = 180  # 3 minutes in seconds


def run_cmd(name, cmd, cwd, timeout):
    start = time.time()
    p = Popen(cmd, cwd=cwd, stdout=PIPE, stderr=PIPE)
    while True:
        # we give up
        if (time.time() - start) >= timeout:
            break

        exit_code = p.poll()
        if exit_code is not None:
            break

    if exit_code != 0:
        o = p.stdout.read()
        if o:
            logging.error(o.strip().decode("UTF-8"))

        e = p.stderr.read()
        if e:
            logging.error(e.strip().decode("UTF-8"))

    assert exit_code == 0


def run_steps(name, path):
    steps = {
        "init": {"type": "run_cmd",
                 "cmd": ["terraform",
                         "init"]},
        "apply": {"type": "run_cmd",
                  "cmd": ["terraform",
                          "apply",
                          "--auto-approve"]},
        "destroy": {"type": "run_cmd",
                    "cmd": ["terraform",
                            "destroy",
                            "--auto-approve"]},
    }

    for step in steps.values():
        run_cmd(name, step["cmd"], path, TIMEOUT)


def test_variants():
    for module in listdir(DISTDIR):
        module_path = join(DISTDIR, module)
        if not isdir(module_path):
            continue

        # yield instructs nose to treat each module as a separate test
        yield run_steps, module, module_path
