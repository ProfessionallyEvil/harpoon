#!/bin/bash
# look for the docker socket if it was (stupidly) mounted.
find "/" 2>&1 | grep -E "(.*\/docker\.sock|^docker\.sock)$"