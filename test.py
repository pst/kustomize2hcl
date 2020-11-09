#!/usr/bin/env python3

import time
import logging
from os import listdir
from os.path import isdir, join
from subprocess import Popen, PIPE, TimeoutExpired
from shutil import unpack_archive

DISTDIR = "modules/"
TIMEOUT = 30  # in seconds


def run_cmd(name, cmd, cwd, timeout):
    p = Popen(cmd, cwd=cwd, stdout=PIPE, stderr=PIPE)
    try:
        outs, errs = p.communicate(timeout=timeout)
    except TimeoutExpired:
        p.kill()
        outs, errs = p.communicate()

    logging.error(outs.strip().decode("UTF-8"))
    logging.error(errs.strip().decode("UTF-8"))

    assert p.returncode == 0


def run_steps(name, path):
    steps = {
        "init": {"type": "run_cmd",
                 "cmd": ["terraform",
                         "init"]},
        "plan": {"type": "run_cmd",
                 "cmd": ["terraform",
                         "plan"]},
        # "apply": {"type": "run_cmd",
        #           "cmd": ["terraform",
        #                   "apply",
        #                   "--auto-approve"]},
        # "destroy": {"type": "run_cmd",
        #             "cmd": ["terraform",
        #                     "destroy",
        #                     "--auto-approve"]},
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
