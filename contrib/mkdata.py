#!/usr/bin/env python3
# coding=utf-8

""" Generate a sample data structure to be used by
    gin-repod."""

from __future__ import print_function
from __future__ import division

import argparse
import os
import subprocess
import shlex
import yaml

tempdir = "tmp"
known_repos = {}


def create_user(user):
    """Create a single user."""
    base = os.path.join("users", user)
    if not os.path.exists(base):
        os.mkdir(base)
    key = os.path.join(base, user + "ssh.key")
    if not os.path.exists(key):
        subprocess.check_call(["ssh-keygen", "-t", "rsa", "-b", "4096",
                               "-C", user, "-f", key, "-P", ""])


def make_repo(repo):
    name = repo["name"]

    if not os.path.exists(tempdir):
        os.makedirs(tempdir)

    target = os.path.join(tempdir, name + ".git")
    if "generate" in repo:
        pwd = os.getcwd()
        cmd = repo["generate"]
        args = shlex.split(cmd)
        args[0] = os.path.join(pwd, args[0])
        os.chdir(tempdir)
        subprocess.check_call(args)
        os.chdir(pwd)
    elif "clone" in repo:
        loc = repo["clone"]
        subprocess.call(["git", "clone", "--bare", loc, target])

    known_repos[name] = target


def create_repo(user, repo):
    """Create a single repo."""
    name = repo["name"]
    base = os.path.join("repos", "git", user)
    path = os.path.join(base, name + ".git")

    if os.path.exists(path):
        return
    elif not os.path.exists(base):
        os.makedirs(base)

    if name not in known_repos:
        make_repo(repo)

    loc = known_repos[name]
    subprocess.call(["git", "clone", "--bare", loc, path])

    # now set up sharing and visibility
    gindir = os.path.join(path, "gin")
    os.mkdir(gindir)
    if repo.get("public") is not None:
        target = os.path.join(gindir, "public")
        os.open(target, flags=os.O_CREAT)
    shared = repo.get("shared") or {}
    for buddy, level in shared.items():
        sharing = os.path.join(gindir, "sharing")
        os.mkdir(sharing)
        target = os.path.join(sharing, buddy)
        with open(target, "w") as fd:
            fd.write(level)


def main():
    """They set us up the main"""
    parser = argparse.ArgumentParser(description="generate sample data dir")
    parser.add_argument("file")
    args = parser.parse_args()

    if not os.path.exists("users"):
        os.mkdir("users")
    if not os.path.exists("repos"):
        os.mkdir("repos")

    doc = yaml.load(open(args.file))
    users = doc['users']
    for user, data in users.items():
        create_user(user)
        repos = data['repos']
        for name, opts in repos.items():
            opts["name"] = name
            create_repo(user, opts)


if __name__ == '__main__':
    main()
