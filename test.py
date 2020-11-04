#!/usr/bin/env python3

import time
import logging
from os import listdir
from os.path import isdir, isfile, join, abspath
from subprocess import Popen, PIPE
from tempfile import TemporaryDirectory
from nose import with_setup
from shutil import unpack_archive

DISTDIR = "modules"
TIMEOUT = 180  # 3 minutes in seconds


def run_cmd(name, cmd, timeout):
    start = time.time()
    p = Popen(cmd, stdout=PIPE, stderr=PIPE)
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
    tfvar_arg = f"--var=path={path}"
    steps = {
        "apply": {"type": "run_cmd",
                          "cmd": ["terraform",
                                  "apply",
                                  "--auto-approve",
                                  tfvar_arg]},
        "destroy": {"type": "run_cmd",
                            "cmd": ["terraform",
                                    "destroy",
                                    "--auto-approve",
                                    tfvar_arg]}
    }

    for step in steps.values():
        run_cmd(name, step["cmd"], TIMEOUT)


def setup():
    run_cmd("init", ["terraform", "init"], TIMEOUT)


def teardown():
    pass


@with_setup(setup, teardown)
def test_variants():
    for module in listdir(DISTDIR):
        module_path = join(DISTDIR, module)
        if not isdir(module_path):
            continue

        # yield instructs nose to treat each module as a separate test
        yield run_steps, module, module_path
